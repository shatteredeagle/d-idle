package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/metrics"
)

// ProcSample is one process at one sample. PID and Start identify it; a change of either between two samples
// is a restart of that process.
type ProcSample struct {
	PID      int    `json:"pid"`
	Start    uint64 `json:"start"`
	State    string `json:"state,omitempty"`
	RSSKB    int64  `json:"rssKB"`
	PSSKB    int64  `json:"pssKB,omitempty"`
	FDs      int    `json:"fds"`
	Threads  int    `json:"threads"`
	CPUTicks uint64 `json:"cpuTicks"`
}

// Sample is one line of samples.jsonl. Absent data is omitted or negative (Progress -1), never guessed.
type Sample struct {
	T         time.Time             `json:"t"`
	HTTP      int                   `json:"http"`     // status of the health call, 0 = unreachable
	HealthOK  bool                  `json:"healthOk"` // reachable and 200
	State     string                `json:"state,omitempty"`
	Paused    bool                  `json:"paused,omitempty"`
	Fatal     bool                  `json:"fatal,omitempty"`
	Stuck     string                `json:"stuck,omitempty"`
	Panics    int                   `json:"panics,omitempty"` // health.panics: recovered panics since the bot started
	Ladder    int                   `json:"ladder"`           // health.ladderStep
	Progress  int64                 `json:"progress"`         // health.progressCounter, -1 when the health body has none
	Level     int                   `json:"level,omitempty"`
	Area      string                `json:"area,omitempty"` // /status runner.character.area (absent in older runs)
	CharXP    int64                 `json:"charXp,omitempty"`
	SaveXP    int64                 `json:"saveXp,omitempty"`    // total experience stored in the .d2s (-save-file), 0 = not read
	SaveLevel int                   `json:"saveLevel,omitempty"` // level stored in the .d2s
	XP        int64                 `json:"xp"`
	Kills     int64                 `json:"kills"`
	Drops     int64                 `json:"drops"`
	Deaths    int64                 `json:"deaths"`
	Errors    int64                 `json:"errors"`
	HasStats  bool                  `json:"hasStats,omitempty"`
	Seq       uint64                `json:"seq"`                // newest event seq seen
	Events    map[string]int        `json:"events,omitempty"`   // events of each kind since the previous sample
	Sentinel  int                   `json:"sentinel,omitempty"` // sentinel "acted" lines since the previous sample
	Procs     map[string]ProcSample `json:"procs,omitempty"`
	Gauges    map[string]float64    `json:"gauges,omitempty"` // scraped from /metrics
	Wine      *WineCount            `json:"wine,omitempty"`   // winedevice.exe of the instance prefix (nil = not sampled)
	Err       string                `json:"err,omitempty"`
}

// ProcSource names the processes to watch. Spec is a pid, "self", or "match:SUBSTR" (the lowest pid whose
// comm or command line contains SUBSTR, re-resolved at every sample so restarts are followed).
type ProcSource struct {
	Name, Spec string
}

func parseProcSpecs(specs []string) ([]ProcSource, error) {
	var out []ProcSource
	for _, s := range specs {
		n, spec, ok := strings.Cut(s, "=")
		if !ok || n == "" || spec == "" {
			return nil, fmt.Errorf("bad -proc %q, want name=PID|self|match:SUBSTR|find:comm=..;cmd=..;env=K=V", s)
		}
		out = append(out, ProcSource{n, spec})
	}
	return out, nil
}

// Collector reads one target. Fields are set before the first Sample.
type Collector struct {
	HTTP         *http.Client
	Base         string // .../api/v1 or .../api/v1/instances/ID
	Flat         bool   // legacy flat routes (events are goals.Event, health is /bot/health)
	Token        string
	MetricsURL   string // "" = none
	Procs        []ProcSource
	ProcRoot     string                       // default /proc
	ProcOverride func() map[string]ProcSample // tests
	SentinelLog  string                       // JSONL path ("" = none)
	WinePrefixes []string                     // WINEPREFIX values whose winedevice.exe are counted (none = no sampling)
	SaveFile     string                       // the character's .d2s ("" = none): saved experience for the XP view
	// Retarget, when set, runs at the start of every Sample (same goroutine) and may move the collector to a new
	// bot address (a managed bot listens on an ephemeral port that changes with every restart).
	Retarget func(ctx context.Context, c *Collector)

	lastSeq   uint64
	sentOff   int64
	sentReady bool
}

func (c *Collector) get(ctx context.Context, path string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
	if err != nil {
		return 0, nil, err
	}
	return c.do(req)
}

func (c *Collector) do(req *http.Request) (int, []byte, error) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return resp.StatusCode, b, err
}

// Resolve picks the instance route: base is a server root (http://host:8420) or an api base.
func Resolve(ctx context.Context, cl *http.Client, token, target, instance string) (*Collector, error) {
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("bad -target %q", target)
	}
	root := strings.TrimRight(target, "/")
	root = strings.TrimSuffix(root, "/api/v1")
	c := &Collector{HTTP: cl, Token: token}
	if instance != "" {
		c.Base = root + "/api/v1/instances/" + instance
		return c, nil
	}
	c.Base = root + "/api/v1"
	code, b, err := c.get(ctx, "/instances")
	if err != nil {
		return nil, fmt.Errorf("target unreachable: %w", err)
	}
	if code == 200 {
		var list []struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(b, &list) == nil && len(list) > 0 {
			c.Base += "/instances/" + list[0].ID
			return c, nil
		}
	}
	c.Flat = true // cmd/d2bot or d2dash without the unified server
	return c, nil
}

func num(m map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		switch v := m[k].(type) {
		case float64:
			return v, true
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func str(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			return s
		}
	}
	return ""
}

// truthy: fatal is a bool or an error text in the health body.
func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x != ""
	}
	return false
}

// Version asks the bot's /status for its build stamp ("d2bot <commit>"); "unknown" when it cannot be read.
func (c *Collector) Version(ctx context.Context) string {
	code, b, err := c.get(ctx, "/status")
	if err != nil || code != 200 {
		return "unknown"
	}
	var m struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(b, &m) != nil || m.Version == "" {
		return "unknown"
	}
	return strings.TrimPrefix(m.Version, "d2bot ")
}

func boolField(m map[string]any, k string) bool { b, _ := m[k].(bool); return b }

// Sample takes one sample stamped now. It never fails: errors are recorded in the sample.
func (c *Collector) Sample(ctx context.Context, now time.Time) Sample {
	s := Sample{T: now, Progress: -1}
	var errs []string
	if c.Retarget != nil {
		c.Retarget(ctx, c)
	}

	hp := "/health"
	if c.Flat {
		hp = "/bot/health"
	}
	code, b, err := c.get(ctx, hp)
	s.HTTP = code
	if err != nil {
		errs = append(errs, "health: "+err.Error())
	} else {
		s.HealthOK = code == 200
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			s.State = str(m, "state")
			s.Paused = boolField(m, "paused") || boolField(m, "selfPaused")
			s.Fatal = truthy(m["fatal"])
			s.Stuck = str(m, "stuckReason")
			if v, ok := num(m, "ladderStep"); ok {
				s.Ladder = int(v)
			}
			if v, ok := num(m, "panics"); ok {
				s.Panics = int(v)
			}
			if v, ok := num(m, "progressCounter"); ok {
				s.Progress = int64(v)
			}
		}
	}

	if code, b, err := c.get(ctx, "/stats"); err == nil && code == 200 {
		var st struct {
			Totals struct {
				Kills, Drops, Deaths, Errors, XP int64
			} `json:"totals"`
		}
		if json.Unmarshal(b, &st) == nil {
			s.HasStats = true
			s.XP, s.Kills, s.Drops, s.Deaths, s.Errors = st.Totals.XP, st.Totals.Kills, st.Totals.Drops, st.Totals.Deaths, st.Totals.Errors
		}
	}
	if code, b, err := c.get(ctx, "/status"); err == nil && code == 200 {
		var st struct {
			Runner struct {
				Character struct {
					Level int
					XP    int64
					Area  string
				} `json:"character"`
			} `json:"runner"`
		}
		if json.Unmarshal(b, &st) == nil {
			c := st.Runner.Character
			s.Level, s.CharXP, s.Area = c.Level, c.XP, c.Area
		}
	}

	readSave(c.SaveFile, &s)
	s.Events, s.Seq = c.newEvents(ctx)
	s.Sentinel = c.sentinelActed()
	s.Procs = c.procs()
	if len(c.WinePrefixes) > 0 {
		root := c.ProcRoot
		if root == "" {
			root = "/proc"
		}
		w := countWineDevices(root, c.WinePrefixes)
		s.Wine = &w
	}
	if c.MetricsURL != "" {
		if g, err := c.scrape(ctx); err != nil {
			errs = append(errs, "metrics: "+err.Error())
		} else {
			s.Gauges = g
		}
	}
	s.Err = strings.Join(errs, "; ")
	return s
}

// newEvents counts events newer than the last seen seq, by kind. The server keeps 1000 per instance; a gap is
// possible on a very busy bot and then the counts are a lower bound (deaths come from /stats anyway).
func (c *Collector) newEvents(ctx context.Context) (map[string]int, uint64) {
	path := fmt.Sprintf("/events?since=%d&limit=1000", c.lastSeq)
	code, b, err := c.get(ctx, path)
	if err != nil || code != 200 {
		return nil, c.lastSeq
	}
	var evs []struct {
		Seq  uint64 `json:"seq"`
		Kind string `json:"kind"`
		Type string `json:"type"` // flat goals.Event
		ID   uint64 `json:"id"`
	}
	if json.Unmarshal(b, &evs) != nil {
		return nil, c.lastSeq
	}
	out := map[string]int{}
	for _, e := range evs {
		seq := e.Seq
		if seq == 0 {
			seq = e.ID
		}
		if seq <= c.lastSeq {
			continue
		}
		c.lastSeq = seq
		k := e.Kind
		if k == "" {
			k = e.Type
		}
		out[k]++
	}
	return out, c.lastSeq
}

// sentinelActed counts new `"kind":"acted"` lines. The first call only records the end of the file, so a log
// that predates the soak does not count.
func (c *Collector) sentinelActed() int {
	if c.SentinelLog == "" {
		return 0
	}
	f, err := os.Open(c.SentinelLog)
	if err != nil {
		return 0
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0
	}
	if !c.sentReady || st.Size() < c.sentOff { // first look, or rotated
		first := !c.sentReady
		c.sentReady = true
		if first {
			c.sentOff = st.Size()
			return 0
		}
		c.sentOff = 0 // rotated: count the new file from its start
	}
	f.Seek(c.sentOff, io.SeekStart)
	n := 0
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if !strings.HasSuffix(line, "\n") { // partial line: read it next time
			break
		}
		c.sentOff += int64(len(line))
		var rec struct{ Kind string }
		if json.Unmarshal([]byte(line), &rec) == nil && rec.Kind == "acted" {
			n++
		}
		if err != nil {
			break
		}
	}
	return n
}

func (c *Collector) procs() map[string]ProcSample {
	if c.ProcOverride != nil {
		return c.ProcOverride()
	}
	if len(c.Procs) == 0 {
		return nil
	}
	root := c.ProcRoot
	if root == "" {
		root = "/proc"
	}
	out := map[string]ProcSample{}
	for _, ps := range c.Procs {
		pid := 0
		switch {
		case ps.Spec == "self":
			pid = os.Getpid()
		case strings.HasPrefix(ps.Spec, "find:"):
			f, err := parseFind(strings.TrimPrefix(ps.Spec, "find:"))
			if err == nil {
				pid = findPID(root, f, os.Getpid())
			}
		case strings.HasPrefix(ps.Spec, "match:"):
			sub := strings.TrimPrefix(ps.Spec, "match:")
			self := os.Getpid()
			for _, p := range metrics.FindPIDs(root, func(comm, cmd string) bool {
				return strings.Contains(comm, sub) || strings.Contains(cmd, sub)
			}) {
				if p != self {
					pid = p
					break
				}
			}
		default:
			pid, _ = strconv.Atoi(ps.Spec)
		}
		if pid <= 0 {
			continue
		}
		p, err := metrics.ReadProcAt(root, pid)
		if err != nil {
			continue
		}
		out[ps.Name] = ProcSample{PID: p.PID, Start: p.Start, State: p.State, RSSKB: p.RSSKB, PSSKB: p.PSSKB,
			FDs: p.FDs, Threads: p.Threads, CPUTicks: p.CPUTicks()}
	}
	return out
}

// scrape reads the bot's Prometheus text and keeps the leak-relevant gauges.
func (c *Collector) scrape(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.MetricsURL, nil)
	if err != nil {
		return nil, err
	}
	code, b, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("HTTP %d", code)
	}
	return parseGauges(string(b)), nil
}

// leakGauges are the unlabelled series the harness tracks for slopes.
var leakGauges = map[string]bool{"go_goroutines": true, "go_threads": true, "process_open_fds": true,
	"process_resident_memory_bytes": true, "go_memstats_heap_inuse_bytes": true, "go_memstats_heap_objects": true}

func parseGauges(text string) map[string]float64 {
	out := map[string]float64{}
	for _, line := range strings.Split(text, "\n") {
		name, val, ok := strings.Cut(line, " ")
		if !ok || !leakGauges[name] {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			out[name] = v
		}
	}
	return out
}
