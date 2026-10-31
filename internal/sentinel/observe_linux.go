package sentinel

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
)

// Observer reads one instance. Every read is non-intrusive: /proc, a read-only /proc/pid/mem, the helper's
// info/hung commands, the instance's own X display (window list, one window capture) and the bot's health URL.
type Observer struct {
	Inst InstanceConf
	Cfg  Config

	proc     *mem.Proc
	procPID  int
	lastTick uint64
	lastAt   time.Time

	win       string
	frameAt   time.Time
	frameHash string
	frameOK   bool

	dlgAt    time.Time
	dialogs  []string
	winNames map[string]winInfo
	http     *http.Client
}

type winInfo struct {
	name string
	w, h int
}

func NewObserver(in InstanceConf, cfg Config) *Observer {
	return &Observer{Inst: in, Cfg: cfg, winNames: map[string]winInfo{}, http: &http.Client{Timeout: 2 * time.Second}}
}

// lockHeld probes the instance lock files without keeping them.
func lockHeld(id string) bool { return launch.LockHeld(id) }

// procStat returns state letter and utime+stime of a pid.
func procStat(pid int) (string, uint64, error) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", 0, err
	}
	s := string(b)
	k := strings.LastIndexByte(s, ')')
	if k < 0 {
		return "", 0, fmt.Errorf("bad stat")
	}
	f := strings.Fields(s[k+1:])
	if len(f) < 13 {
		return "", 0, fmt.Errorf("short stat")
	}
	u, _ := strconv.ParseUint(f[11], 10, 64)
	sy, _ := strconv.ParseUint(f[12], 10, 64)
	return f[0], u + sy, nil
}

func (ob *Observer) Observe(now time.Time) Observation {
	o := Observation{Time: now, Instance: ob.Inst.ID}
	o.LockHeld = lockHeld(ob.Inst.ID)
	pid, err := mem.FindGameFor(ob.Inst.ID)
	if err == nil {
		st, ticks, err := procStat(pid)
		if err == nil {
			o.ProcAlive, o.PID, o.ProcState, o.CPUTicks = true, pid, st, ticks
			if ob.lastAt.IsZero() || pid != ob.procPID {
				ob.lastTick, ob.lastAt = ticks, now
			} else if dt := now.Sub(ob.lastAt).Seconds(); dt >= 1 {
				o.CPUPct = float64(ticks-ob.lastTick) / 100 / dt * 100
				ob.lastTick, ob.lastAt = ticks, now
			}
		}
	}
	if !o.ProcAlive {
		if ob.proc != nil {
			ob.proc.Close()
			ob.proc = nil
		}
		ob.win, ob.frameHash, ob.frameOK = "", "", false
	}
	if o.ProcAlive {
		ob.readMem(&o)
		ob.frame(&o, now)
	}
	ob.helper(&o)
	if o.ProcAlive || ob.Inst.ID != "i0" {
		ob.dialogScan(&o, now)
	}
	if ob.Inst.BotURL != "" {
		bh := ob.botHealth(now)
		o.Bot = &bh
	}
	return o
}

func (ob *Observer) readMem(o *Observation) {
	if ob.proc == nil || ob.procPID != o.PID {
		if ob.proc != nil {
			ob.proc.Close()
			ob.proc = nil
		}
		p, err := mem.Open(o.PID)
		if err != nil {
			o.MemErr = err.Error()
			return
		}
		ob.proc, ob.procPID = p, o.PID
	}
	r := d2.New(ob.proc)
	st := r.State()
	o.State = st.String()
	o.MemOK = true
	switch st {
	case d2.StateInGame:
		p, err := r.Player()
		if err != nil {
			o.State = "loading"
			return
		}
		o.Mode, o.Life, o.MaxLife, o.X, o.Y, o.Area, o.Level = p.Mode, p.Life, p.MaxLife, p.X, p.Y, p.Area, p.Level
		o.Exp, o.Gold, o.InTown, o.CharName = p.Exp, p.Gold, p.InTown(), p.Name
		if ui, err := r.UI(); err == nil {
			o.EscMenu = ui.Open(d2.UIEscMenu)
			o.Panels = openPanels(ui)
		}
	case d2.StateMenu:
		if cs, err := launch.ReadControls(ob.proc); err == nil {
			o.MenuScr = string(launch.Identify(cs))
		}
	}
}

func xenv(display string) []string {
	return append(os.Environ(), "DISPLAY="+display, "OMP_NUM_THREADS=1", "MAGICK_THREAD_LIMIT=1", "OMP_WAIT_POLICY=passive")
}

func (ob *Observer) xcmd(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, name, args...)
	c.Env = xenv(ob.Inst.DisplayName())
	return c.Output()
}

// frame hashes the game window of the instance (import -window <id>; never the root window, never i0's desktop).
func (ob *Observer) frame(o *Observation, now time.Time) {
	if ob.Inst.ID == "i0" {
		return // i0 lives on the user's real desktop; no capture there
	}
	if now.Sub(ob.frameAt) >= ob.Cfg.FrameEvery.D() || ob.frameAt.IsZero() {
		ob.frameAt = now
		if ob.win == "" {
			if out, err := ob.xcmd(3*time.Second, "xdotool", "search", "--name", "^Diablo II$"); err == nil {
				if f := strings.Fields(string(out)); len(f) > 0 {
					ob.win = f[0]
				}
			}
		}
		ob.frameOK = false
		if ob.win != "" {
			out, err := ob.xcmd(5*time.Second, "import", "-window", ob.win, "-depth", "8", "rgb:-")
			if err == nil && len(out) > 1000 {
				h := sha256.Sum256(out)
				ob.frameHash, ob.frameOK = hex.EncodeToString(h[:8]), true
			} else {
				ob.win = ""
			}
		}
	}
	o.FrameOK, o.FrameHash = ob.frameOK, ob.frameHash
}

// helperCmd sends one line to the d2input helper and returns the reply line.
func helperCmd(addr, cmd string, timeout time.Duration) (string, error) {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return "", err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(timeout))
	if _, err := fmt.Fprintln(c, cmd); err != nil {
		return "", err
	}
	line, err := bufio.NewReader(c).ReadString('\n')
	return strings.TrimSpace(line), err
}

func (ob *Observer) helper(o *Observation) {
	addr := ob.Inst.Helper()
	r, err := helperCmd(addr, "info", 1500*time.Millisecond)
	if err != nil && r == "" {
		o.HelperUp = false
		return
	}
	o.HelperUp = strings.HasPrefix(r, "ok") || strings.HasPrefix(r, "err")
	o.HelperWindow = strings.HasPrefix(r, "ok")
	if o.HelperWindow {
		if h, err := helperCmd(addr, "hung", 3*time.Second); err == nil {
			o.Hung = strings.HasSuffix(h, "hung")
		} else {
			o.Hung = true // the helper could not get an answer about the window in time
		}
	}
}

var dialogRe = regexp.MustCompile(`(?i)critical error|error|only one copy|assert|exception|crash|blizzard`)
var ignoredWin = map[string]bool{"": true, "Diablo II": true, "Default IME": true, "Input": true, "Xalia event window": true, "MSCTFIME UI": true}

// dialogScan lists the windows of the instance's own display and flags unexpected ones, plus crash-reporter
// processes of this instance. Window names are cached by id.
func (ob *Observer) dialogScan(o *Observation, now time.Time) {
	if ob.Inst.ID != "i0" && now.Sub(ob.dlgAt) >= 4*time.Second {
		ob.dlgAt = now
		ob.dialogs = nil
		if out, err := ob.xcmd(3*time.Second, "xdotool", "search", "--name", "."); err == nil {
			seen := map[string]bool{}
			for _, id := range strings.Fields(string(out)) {
				seen[id] = true
				wi, ok := ob.winNames[id]
				if !ok {
					if nm, err := ob.xcmd(2*time.Second, "xdotool", "getwindowname", id); err == nil {
						wi.name = strings.TrimSpace(string(nm))
					}
					if g, err := ob.xcmd(2*time.Second, "xdotool", "getwindowgeometry", "--shell", id); err == nil {
						for _, l := range strings.Split(string(g), "\n") {
							if v, ok := strings.CutPrefix(l, "WIDTH="); ok {
								wi.w, _ = strconv.Atoi(v)
							}
							if v, ok := strings.CutPrefix(l, "HEIGHT="); ok {
								wi.h, _ = strconv.Atoi(v)
							}
						}
					}
					ob.winNames[id] = wi
				}
				if ignoredWin[wi.name] || strings.HasPrefix(strings.ToLower(wi.name), "d2input") || (wi.w <= 2 && wi.h <= 2) {
					continue
				}
				if dialogRe.MatchString(wi.name) || (wi.w >= 100 && wi.h >= 60) {
					ob.dialogs = append(ob.dialogs, fmt.Sprintf("%s [%s %dx%d]", wi.name, id, wi.w, wi.h))
				}
			}
			for id := range ob.winNames {
				if !seen[id] {
					delete(ob.winNames, id)
				}
			}
		}
		for _, p := range crashProcs(ob.Inst.ID) {
			ob.dialogs = append(ob.dialogs, p)
		}
	}
	o.Dialogs = ob.dialogs
}

// crashProcs finds BlizzardError.exe processes that belong to the instance.
func crashProcs(id string) []string {
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		b, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err != nil || !bytes.Contains(bytes.ToLower(b), []byte("blizzarderror")) {
			continue
		}
		if m := mem.ProcInstance(pid); m == id || (id == "i0" && m == "") {
			out = append(out, "BlizzardError.exe pid "+e.Name())
		}
	}
	return out
}

func (ob *Observer) botHealth(now time.Time) BotHealth {
	resp, err := ob.http.Get(ob.Inst.BotURL)
	if err != nil {
		return BotHealth{Err: err.Error(), SinceProgress: -1}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 500 && len(b) == 0 {
		return BotHealth{Err: resp.Status, SinceProgress: -1}
	}
	h := ParseBotHealth(b, now)
	if resp.StatusCode >= 500 && h.State == "" {
		h.Err = resp.Status
	}
	return h
}

// openPanels lists the UI flag indexes of panels that block clicks (the table in internal/d2/ui.go), without the Esc
// menu, which Observation.EscMenu reports separately. HUD flags such as 6/7 (stat / skill point buttons) and 0x11 never count.
func openPanels(ui d2.UIState) []int {
	var out []int
	for _, i := range ui.Indexes(d2.UIPanel) {
		if i != d2.UIEscMenu {
			out = append(out, i)
		}
	}
	return out
}
