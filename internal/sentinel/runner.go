package sentinel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Mode says what the sentinel may do.
type Mode struct {
	Act      bool            `json:"act"`                    // --act: all rules may act
	ActRules map[string]bool `json:"actRules"`               // --act-rules a,b: only these
	ActInst  map[string]bool `json:"actInstances,omitempty"` // --act-instances i5: acting only on these
}

// MayAct reports whether the rule's decisions are executed (otherwise they are logged as "would").
func (m Mode) MayAct(rule, inst string) bool {
	if len(m.ActInst) > 0 && !m.ActInst[inst] {
		return false
	}
	if len(m.ActRules) > 0 {
		return m.ActRules[rule]
	}
	return m.Act
}

func (m Mode) String() string {
	if len(m.ActInst) > 0 {
		var r []string
		for k := range m.ActInst {
			r = append(r, k)
		}
		sort.Strings(r)
		m.ActInst = nil
		return m.String() + "@" + strings.Join(r, ",")
	}
	if len(m.ActRules) > 0 {
		var r []string
		for k := range m.ActRules {
			r = append(r, k)
		}
		sort.Strings(r)
		return "act:" + strings.Join(r, ",")
	}
	if m.Act {
		return "act:all"
	}
	return "observe-only"
}

// Record is one JSONL log line. Kind: obs | would | acted | skipped | start | error.
type Record struct {
	Time     time.Time    `json:"time"`
	Kind     string       `json:"kind"`
	Instance string       `json:"instance,omitempty"`
	Rule     string       `json:"rule,omitempty"`
	Action   Action       `json:"action,omitempty"`
	Evidence string       `json:"evidence,omitempty"`
	Attempt  int          `json:"attempt,omitempty"`
	Step     int          `json:"step,omitempty"`
	Result   *Result      `json:"result,omitempty"`
	Obs      *Observation `json:"obs,omitempty"`
	Note     string       `json:"note,omitempty"`
}

// Runner drives one goroutine per instance: observe, evaluate, act (when allowed), log.
type Runner struct {
	Cfg     Config
	Mode    Mode
	Enabled map[string]bool // instance ids (empty = all configured, minus enabled:false)
	Engine  *Engine
	Actor   Actor
	Observe func(in InstanceConf) func(now time.Time) Observation
	Now     func() time.Time
	LogW    func(Record)

	mu      sync.Mutex
	status  map[string]*InstStatus
	recent  []Record
	onEvent func(Record)
}

// InstStatus is the per-instance health shown by the API.
type InstStatus struct {
	ID         string       `json:"id"`
	Agent      string       `json:"agent,omitempty"`
	Character  string       `json:"character,omitempty"`
	Health     string       `json:"health"` // ok | attention | down | unknown
	Line       string       `json:"line"`
	Obs        *Observation `json:"obs,omitempty"`
	Rules      []RuleStatus `json:"rules,omitempty"`
	LastAction *Record      `json:"lastAction,omitempty"`
	Acting     string       `json:"acting,omitempty"` // action currently running
	key        string
	lastLogged string
	lastObsLog time.Time
}

func NewRunner(cfg Config, mode Mode, actor Actor, w *os.File) *Runner {
	r := &Runner{Cfg: cfg, Mode: mode, Engine: NewEngine(cfg), Actor: actor, Now: time.Now, status: map[string]*InstStatus{}}
	if w != nil {
		var mu sync.Mutex
		r.LogW = func(rec Record) {
			b, _ := json.Marshal(rec)
			mu.Lock()
			w.Write(append(b, '\n'))
			mu.Unlock()
		}
	}
	return r
}

func (r *Runner) active() []InstanceConf {
	var out []InstanceConf
	for _, in := range r.Cfg.Instances {
		if !in.On() {
			continue
		}
		if len(r.Enabled) > 0 && !r.Enabled[in.ID] {
			continue
		}
		out = append(out, in)
	}
	return out
}

func (r *Runner) log(rec Record) {
	if rec.Time.IsZero() {
		rec.Time = r.Now()
	}
	if r.LogW != nil {
		r.LogW(rec)
	}
	if rec.Kind != "obs" {
		r.mu.Lock()
		r.recent = append(r.recent, rec)
		if len(r.recent) > 100 {
			r.recent = r.recent[len(r.recent)-100:]
		}
		r.mu.Unlock()
		if r.onEvent != nil {
			r.onEvent(rec)
		}
	}
}

// Run starts the per-instance loops and blocks until ctx ends.
func (r *Runner) Run(ctx context.Context) {
	ins := r.active()
	var ids []string
	for _, in := range ins {
		ids = append(ids, in.ID)
		r.Engine.State(in.ID) // pre-create: per-instance goroutines then touch only their own state
		r.mu.Lock()
		r.status[in.ID] = &InstStatus{ID: in.ID, Agent: in.Agent, Character: in.Character, Health: "unknown"}
		r.mu.Unlock()
	}
	r.log(Record{Kind: "start", Note: fmt.Sprintf("mode=%s instances=%s", r.Mode, strings.Join(ids, ","))})
	var wg sync.WaitGroup
	for _, in := range ins {
		wg.Add(1)
		go func(in InstanceConf) {
			defer wg.Done()
			r.loop(ctx, in)
		}(in)
	}
	wg.Wait()
}

func (r *Runner) loop(ctx context.Context, in InstanceConf) {
	obs := r.Observe(in)
	t := time.NewTicker(r.Cfg.Interval.D())
	defer t.Stop()
	for {
		r.Tick(ctx, in, obs(r.Now()))
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick evaluates one observation (exported for tests).
func (r *Runner) Tick(ctx context.Context, in InstanceConf, o Observation) {
	decisions := r.Engine.Step(o)
	st := r.summarize(in, &o)
	for _, d := range decisions {
		rec := Record{Time: o.Time, Instance: o.Instance, Rule: d.Rule, Action: d.Action, Evidence: d.Evidence, Attempt: d.Attempt, Step: d.Step}
		if !r.Mode.MayAct(d.Rule, d.Instance) {
			rec.Kind = "would"
			r.log(rec)
			r.setLast(in.ID, rec)
			continue
		}
		r.mu.Lock()
		r.status[in.ID].Acting = string(d.Action)
		r.mu.Unlock()
		res := r.Actor.Do(ctx, in, d.Action, d)
		rec.Kind = "acted"
		if !res.OK && strings.Contains(res.Detail, errLocked.Error()) {
			rec.Kind = "skipped"
		}
		rec.Result = &res
		rec.Time = r.Now()
		r.log(rec)
		r.setLast(in.ID, rec)
		r.mu.Lock()
		r.status[in.ID].Acting = ""
		r.mu.Unlock()
		// The action took time; the engine's settle window starts now.
		r.Engine.State(in.ID).lastAction = r.Now()
	}
	// Observation records: on a change of the summary line, and every LogEvery.
	r.mu.Lock()
	line := st.key
	logObs := line != st.lastLogged || o.Time.Sub(st.lastObsLog) >= r.Cfg.LogEvery.D()
	if logObs {
		st.lastLogged, st.lastObsLog = line, o.Time
	}
	r.mu.Unlock()
	if logObs {
		oc := o
		r.log(Record{Time: o.Time, Kind: "obs", Instance: in.ID, Obs: &oc, Note: st.Line})
	}
}

func (r *Runner) setLast(id string, rec Record) {
	r.mu.Lock()
	r.status[id].LastAction = &rec
	r.mu.Unlock()
}

// summarize updates and returns the instance status.
func (r *Runner) summarize(in InstanceConf, o *Observation) *InstStatus {
	rules := r.Engine.Status(in.ID, o.Time)
	var active, names []string
	for _, rs := range rules {
		if rs.Active {
			active = append(active, fmt.Sprintf("%s %.0fs", rs.Rule, rs.ForSec))
			names = append(names, rs.Rule)
		}
	}
	health, line := "ok", ""
	switch {
	case !o.ProcAlive:
		health, line = "down", "no game process"
	case len(active) > 0:
		health = "attention"
	}
	if o.ProcAlive {
		switch {
		case o.State == "ingame" && o.Dead():
			line = "ingame DEAD"
		case o.State == "ingame":
			line = fmt.Sprintf("ingame %s L%d area %d life %d/%d", o.CharName, o.Level, o.Area, o.Life, o.MaxLife)
			if o.EscMenu {
				line += " ESC-MENU"
			}
			if len(o.Panels) > 0 {
				line += " panels=" + joinInts(o.Panels)
			}
		default:
			line = o.State + " " + o.MenuScr
		}
	}
	key := line + "|" + strings.Join(names, ",")
	if len(active) > 0 {
		line += " | " + strings.Join(active, ", ")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.status[in.ID]
	if st == nil {
		st = &InstStatus{ID: in.ID}
		r.status[in.ID] = st
	}
	oc := *o
	st.Obs, st.Rules, st.Health, st.Line, st.key = &oc, rules, health, line, key
	return st
}

// Snapshot is the /api/sentinel document.
type Snapshot struct {
	Time      time.Time    `json:"time"`
	Mode      string       `json:"mode"`
	Summary   string       `json:"summary"` // one line for d2watch
	Instances []InstStatus `json:"instances"`
	Recent    []Record     `json:"recent"`
	Thresh    Thresholds   `json:"thresholds"`
}

func (r *Runner) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := Snapshot{Time: r.Now(), Mode: r.Mode.String(), Thresh: r.Cfg.Th}
	var ids []string
	for id := range r.status {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var parts []string
	for _, id := range ids {
		st := *r.status[id]
		s.Instances = append(s.Instances, st)
		p := id + " " + strings.ToUpper(st.Health)
		if st.Acting != "" {
			p += " ACTING " + st.Acting
		}
		parts = append(parts, p)
	}
	s.Summary = "sentinel[" + s.Mode + "] " + strings.Join(parts, " | ")
	n := len(r.recent)
	if n > 50 {
		n = 50
	}
	for i := len(r.recent) - 1; i >= len(r.recent)-n; i-- {
		s.Recent = append(s.Recent, r.recent[i])
	}
	return s
}
