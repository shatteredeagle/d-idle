package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/viewport"
)

const (
	trailKeep   = 4 * time.Second
	apiIdle     = 5 * time.Second // metadata sampling stops when nobody polled the API for this long
	mouseTick   = 100 * time.Millisecond
	playerEvery = 500 * time.Millisecond
)

type pt struct {
	X, Y int
	T    time.Time
}

// Inst is one game instance. Fields under mu are shared between the discovery, sampler, capture and HTTP goroutines.
type Inst struct {
	ID      string
	N       int
	Agent   string
	Display string
	Port    int

	vp *viewport.Stream // lazy shared capture of the game window

	mu       sync.Mutex
	pid      int
	win      string
	helperUp bool

	// sampler state (only the sampler goroutine touches proc/rd; meta is read under mu)
	proc     *mem.Proc
	rd       *d2.Reader
	procPID  int
	retryAt  time.Time
	lastPl   time.Time
	cpuPID   int
	cpuTicks uint64
	cpuAt    time.Time
	meta     meta
}

type meta struct {
	Mem       string // ok | denied | none
	MemErr    string
	State     string
	Name      string
	Class     string
	Level     int
	Area      int
	Life, Max int
	Mana, MMx int
	Mouse     pt
	HaveMouse bool
	Trail     []pt
	CPU       float64
}

type registry struct {
	conf string
	fps  float64
	capt viewport.Capturer
	lim  *viewport.Limiter

	viewers viewerLimit // open MJPEG streams, all instances

	mu    sync.Mutex
	insts map[string]*Inst

	apiAt atomic.Int64 // unix nano of the last API poll
}

func newRegistry(conf string, fps float64, conc int, capt viewport.Capturer) *registry {
	r := &registry{conf: conf, fps: fps, capt: capt, lim: viewport.NewLimiter(conc), insts: map[string]*Inst{}}
	r.viewers.max = 24
	return r
}

func (r *registry) run() {
	r.discover()
	go r.sampler()
	for range time.Tick(3 * time.Second) {
		r.discover()
	}
}

func readConf(path string) map[string]string { // instance id -> agent name
	out := map[string]string{"i0": "testy"}
	f, err := os.Open(path)
	if err != nil {
		return map[string]string{"i1": "nav", "i2": "combat", "i3": "town", "i4": "bot", "i5": "multi", "i0": "main"}
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	agents := map[string][]string{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		fl := strings.Fields(line)
		if len(fl) >= 2 {
			agents[fl[1]] = append(agents[fl[1]], fl[0])
		}
	}
	for id, as := range agents {
		out[id] = strings.Join(as, "/")
	}
	out["i0"] = "main"
	return out
}

// xenv returns the environment for X tools of an instance. i0 lives on the user's real desktop (:1, XWayland,
// needs its xauth file); the others are private Xvfb displays.
func xenv(n int) []string {
	env := append(os.Environ(), "DISPLAY="+displayOf(n))
	if n == 0 {
		if m, _ := filepath.Glob("/run/user/" + strconv.Itoa(os.Getuid()) + "/xauth_*"); len(m) > 0 {
			env = append(env, "XAUTHORITY="+m[0])
		}
	}
	return env
}

func displayOf(n int) string {
	if n == 0 {
		return ":1"
	}
	return ":" + strconv.Itoa(100+n)
}

// discover finds running games (same markers as tools/game.sh and internal/mem) and updates the tile set.
func (r *registry) discover() {
	agents := readConf(r.conf)
	byID := map[string]int{}
	for _, pid := range mem.FindGamesFor("") {
		id := mem.ProcInstance(pid)
		if id == "" {
			id = mem.DefaultInstance
		}
		if _, dup := byID[id]; !dup {
			byID[id] = pid
		}
	}
	r.mu.Lock()
	for id := range r.insts {
		if _, ok := byID[id]; !ok {
			in := r.insts[id]
			in.mu.Lock()
			in.pid, in.win = 0, ""
			in.mu.Unlock()
			in.vp.Close()
			delete(r.insts, id)
		}
	}
	var fresh []*Inst
	for id, pid := range byID {
		n, err := strconv.Atoi(strings.TrimPrefix(id, "i"))
		if err != nil || !strings.HasPrefix(id, "i") {
			continue
		}
		in := r.insts[id]
		if in == nil {
			in = &Inst{ID: id, N: n, Display: displayOf(n), Port: 47012 + n}
			in.vp = viewport.New(viewport.Config{
				Capturer: r.capt, FPS: r.fps, Quality: 65, Limiter: r.lim, Idle: 3 * time.Second,
				Target: in.target,
				// window gone or game restarted: rediscover on the next discovery tick
				OnError: func(error) { in.mu.Lock(); in.win = ""; in.mu.Unlock() },
			})
			r.insts[id] = in
		}
		in.mu.Lock()
		in.Agent = agents[id]
		in.pid = pid
		need := in.win == ""
		in.mu.Unlock()
		if need {
			fresh = append(fresh, in)
		}
	}
	all := make([]*Inst, 0, len(r.insts))
	for _, in := range r.insts {
		all = append(all, in)
	}
	r.mu.Unlock()

	for _, in := range fresh {
		if w := findWindow(in); w != "" {
			in.mu.Lock()
			in.win = w
			in.mu.Unlock()
		}
	}
	for _, in := range all {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(in.Port), 150*time.Millisecond)
		if err == nil {
			c.Close()
		}
		in.mu.Lock()
		in.helperUp = err == nil
		in.mu.Unlock()
	}
}

// findWindow returns the id of the game window on the instance's display (never the root window).
func findWindow(in *Inst) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "xdotool", "search", "--name", "^Diablo II$")
	cmd.Env = xenv(in.N)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	f := strings.Fields(string(out))
	if len(f) == 0 {
		return ""
	}
	if _, err := strconv.ParseUint(f[0], 10, 64); err != nil {
		return ""
	}
	return f[0]
}

// ---- capture (internal/viewport does the work) ----

// target is the viewport.Config.Target of the instance: the game window on its display, never the root window.
func (in *Inst) target() (viewport.Spec, error) {
	in.mu.Lock()
	win := in.win
	in.mu.Unlock()
	if win == "" {
		return viewport.Spec{}, viewport.ErrNoTarget
	}
	return viewport.Spec{Display: displayOf(in.N), Env: xenv(in.N), Window: win}, nil
}

func (r *registry) get(path, prefix, suffix string) *Inst {
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.insts[id]
}

func (r *registry) handleFrame(w http.ResponseWriter, req *http.Request) {
	in := r.get(req.URL.Path, "/frame/", ".jpg")
	if in == nil {
		fail(w, 404, "no such instance")
		return
	}
	viewport.ServeFrame(in.vp, 250*time.Millisecond, w, req)
}

func (r *registry) handleStream(w http.ResponseWriter, req *http.Request) {
	in := r.get(req.URL.Path, "/stream/", ".mjpg")
	if in == nil {
		fail(w, 404, "no such instance")
		return
	}
	serveStream(in.vp, &r.viewers, in.ID, w, req)
}

// ---- metadata sampler (read-only game memory) ----

func (r *registry) sampler() {
	for range time.Tick(mouseTick) {
		if time.Since(time.Unix(0, r.apiAt.Load())) > apiIdle {
			continue
		}
		r.mu.Lock()
		list := make([]*Inst, 0, len(r.insts))
		for _, in := range r.insts {
			list = append(list, in)
		}
		r.mu.Unlock()
		for _, in := range list {
			in.sample()
		}
	}
}

func (in *Inst) sample() {
	in.mu.Lock()
	pid := in.pid
	in.mu.Unlock()
	if pid == 0 {
		return
	}
	in.sampleCPU(pid)
	if in.proc != nil && (in.procPID != pid || !in.proc.Alive()) {
		in.proc.Close()
		in.proc, in.rd = nil, nil
	}
	now := time.Now()
	if in.proc == nil {
		if now.Before(in.retryAt) {
			return
		}
		p, err := mem.Open(pid)
		if err == nil {
			err = mem.VerifyBuild(p)
			if err != nil {
				p.Close()
			}
		}
		if err != nil {
			in.retryAt = now.Add(5 * time.Second)
			in.mu.Lock()
			in.meta.Mem, in.meta.MemErr = "denied", err.Error()
			in.mu.Unlock()
			return
		}
		in.proc, in.rd, in.procPID = p, d2.New(p), pid
	}
	m, merr := in.rd.Mouse()
	var pl *d2.Player
	var st d2.State
	doPlayer := now.Sub(in.lastPl) >= playerEvery
	if doPlayer {
		in.lastPl = now
		st = in.rd.State()
		if st == d2.StateInGame {
			pl, _ = in.rd.Player()
		}
	}
	in.mu.Lock()
	defer in.mu.Unlock()
	in.meta.Mem, in.meta.MemErr = "ok", ""
	if merr != nil {
		in.meta.Mem, in.meta.MemErr = "denied", merr.Error()
		in.proc.Close()
		in.proc, in.rd = nil, nil
		in.retryAt = now.Add(5 * time.Second)
		return
	}
	if m.X >= 0 && m.X < 4000 && m.Y >= 0 && m.Y < 4000 {
		cur := pt{m.X, m.Y, now}
		in.meta.Mouse, in.meta.HaveMouse = cur, true
		if n := len(in.meta.Trail); n == 0 || in.meta.Trail[n-1].X != cur.X || in.meta.Trail[n-1].Y != cur.Y {
			in.meta.Trail = append(in.meta.Trail, cur)
		}
		i := 0
		for i < len(in.meta.Trail) && now.Sub(in.meta.Trail[i].T) > trailKeep {
			i++
		}
		in.meta.Trail = append([]pt(nil), in.meta.Trail[i:]...)
	}
	if doPlayer {
		in.meta.State = st.String()
		if pl != nil {
			in.meta.Name, in.meta.Class, in.meta.Level, in.meta.Area = pl.Name, pl.Class.String(), pl.Level, pl.Area
			in.meta.Life, in.meta.Max, in.meta.Mana, in.meta.MMx = pl.Life, pl.MaxLife, pl.Mana, pl.MaxMana
		} else if st == d2.StateMenu {
			in.meta.Name, in.meta.Level, in.meta.Area, in.meta.Max, in.meta.MMx = "", 0, 0, 0, 0
		}
	}
}

// sampleCPU updates the Game.exe CPU percent (of one core) about once per second from /proc/<pid>/stat.
func (in *Inst) sampleCPU(pid int) {
	now := time.Now()
	if in.cpuPID == pid && now.Sub(in.cpuAt) < time.Second {
		return
	}
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return
	}
	s := string(b)
	f := strings.Fields(s[strings.LastIndexByte(s, ')')+2:])
	if len(f) < 13 {
		return
	}
	u, _ := strconv.ParseUint(f[11], 10, 64)
	k, _ := strconv.ParseUint(f[12], 10, 64)
	ticks := u + k
	if in.cpuPID == pid && now.After(in.cpuAt) {
		pct := float64(ticks-in.cpuTicks) / 100 / now.Sub(in.cpuAt).Seconds() * 100
		in.mu.Lock()
		in.meta.CPU = pct
		in.mu.Unlock()
	}
	in.cpuPID, in.cpuTicks, in.cpuAt = pid, ticks, now
}

// ---- API ----

type apiPoint struct {
	X   int `json:"x"`
	Y   int `json:"y"`
	Age int `json:"age"` // ms before "now"
}

type apiInst struct {
	ID       string     `json:"id"`
	Agent    string     `json:"agent"`
	Display  string     `json:"display"`
	PID      int        `json:"pid"`
	Window   bool       `json:"window"`
	Helper   bool       `json:"helper"`
	Mem      string     `json:"mem"`
	MemErr   string     `json:"memErr,omitempty"`
	State    string     `json:"state,omitempty"`
	Char     string     `json:"character,omitempty"`
	Class    string     `json:"class,omitempty"`
	Level    int        `json:"level"`
	Area     int        `json:"area"`
	AreaName string     `json:"areaName,omitempty"`
	Life     int        `json:"life"`
	MaxLife  int        `json:"maxLife"`
	Mana     int        `json:"mana"`
	MaxMana  int        `json:"maxMana"`
	Mouse    *apiPoint  `json:"mouse,omitempty"`
	Trail    []apiPoint `json:"trail"`
	CPU      float64    `json:"cpu"`
	FrameAge int        `json:"frameAgeMs"`
	CapErr   string     `json:"captureError,omitempty"`
}

func (r *registry) handleAPI(w http.ResponseWriter, req *http.Request) {
	now := time.Now()
	r.apiAt.Store(now.UnixNano())
	r.mu.Lock()
	list := make([]*Inst, 0, len(r.insts))
	for _, in := range r.insts {
		list = append(list, in)
	}
	r.mu.Unlock()
	sort.Slice(list, func(i, j int) bool { return list[i].N < list[j].N })
	out := make([]apiInst, 0, len(list))
	for _, in := range list {
		in.mu.Lock()
		m := in.meta
		a := apiInst{ID: in.ID, Agent: in.Agent, Display: in.Display, PID: in.pid, Window: in.win != "", Helper: in.helperUp,
			Mem: m.Mem, MemErr: m.MemErr, State: m.State, Char: m.Name, Class: m.Class, Level: m.Level, Area: m.Area,
			AreaName: areaName(m.Area), Life: m.Life, MaxLife: m.Max, Mana: m.Mana, MaxMana: m.MMx, CPU: m.CPU,
			Trail: []apiPoint{}, FrameAge: -1}
		st := in.vp.Status()
		a.CapErr = st.Err
		if !st.FrameAt.IsZero() {
			a.FrameAge = int(now.Sub(st.FrameAt) / time.Millisecond)
		}
		if a.Mem == "" {
			a.Mem = "pending"
		}
		if m.HaveMouse {
			a.Mouse = &apiPoint{m.Mouse.X, m.Mouse.Y, int(now.Sub(m.Mouse.T) / time.Millisecond)}
		}
		for _, p := range m.Trail {
			a.Trail = append(a.Trail, apiPoint{p.X, p.Y, int(now.Sub(p.T) / time.Millisecond)})
		}
		in.mu.Unlock()
		out = append(out, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"instances": out})
}
