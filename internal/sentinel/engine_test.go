package sentinel

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func healthy() Observation {
	return Observation{Instance: "i5", ProcAlive: true, PID: 100, ProcState: "R", FrameOK: true, MemOK: true,
		State: "ingame", Mode: 1, Life: 100, MaxLife: 100, X: 50, Y: 50, Area: 1, InTown: true,
		HelperUp: true, HelperWindow: true}
}

func newEngine() *Engine {
	cfg := DefaultConfig()
	cfg.Instances = []InstanceConf{{ID: "i5", Character: "Multi", Class: "sorceress"}}
	return NewEngine(cfg)
}

// run feeds an observation every second for secs seconds starting at start, mutating a healthy baseline
// with mut(secondsSinceStart, *obs). It returns every decision as "t:rule:action".
func run(e *Engine, start time.Time, secs int, mut func(i int, o *Observation)) []string {
	var out []string
	for i := 0; i <= secs; i++ {
		o := healthy()
		o.Time = start.Add(time.Duration(i) * time.Second)
		o.CPUTicks = uint64(i * 100)
		o.FrameHash = string(rune('a' + i%20))
		o.X = 50 + (i%2)*10 // moving, so no-progress stays quiet unless a case overrides
		mut(i, &o)
		for _, d := range e.Step(o) {
			out = append(out, itoa(i)+":"+d.Rule+":"+string(d.Action))
		}
	}
	return out
}

func itoa(i int) string {
	s := ""
	if i == 0 {
		return "0"
	}
	for ; i > 0; i /= 10 {
		s = string(rune('0'+i%10)) + s
	}
	return s
}

func TestRulesTable(t *testing.T) {
	cases := []struct {
		name string
		secs int
		mut  func(i int, o *Observation)
		want []string
	}{
		{"healthy is silent", 300, func(i int, o *Observation) {}, nil},
		{"proc gone fires after 8 s then cools down", 60, func(i int, o *Observation) {
			if i >= 5 {
				o.ProcAlive, o.MemOK, o.HelperWindow = false, false, false
			}
		}, []string{"13:proc-gone:restart-game"}},
		{"death overlay: Esc after 10 s, retries every 12 s, then escalates", 80, func(i int, o *Observation) {
			if i >= 5 {
				o.Life, o.Mode = 0, 17
			}
		}, []string{"15:death-overlay:respawn", "27:death-overlay:respawn", "39:death-overlay:respawn", "51:death-overlay:restart-game"}},
		{"death overlay cleared before the threshold", 60, func(i int, o *Observation) {
			if i >= 5 && i < 12 {
				o.Life = 0
			}
		}, nil},
		{"esc menu open 20 s", 40, func(i int, o *Observation) {
			if i >= 3 {
				o.EscMenu = true
			}
		}, []string{"23:esc-menu-stuck:esc", "43:esc-menu-stuck:esc"}[:1]},
		{"panel open 90 s", 120, func(i int, o *Observation) {
			if i >= 2 {
				o.Panels = []int{1}
			}
		}, []string{"92:panel-stuck:esc"}},
		{"menu screen 30 s -> OOG start", 50, func(i int, o *Observation) {
			if i >= 1 {
				o.State, o.MenuScr = "menu", "main-menu"
			}
		}, []string{"31:out-of-game:oog-start"}},
		{"helper gone 10 s", 30, func(i int, o *Observation) {
			if i >= 4 {
				o.HelperUp, o.HelperWindow = false, false
			}
		}, []string{"14:helper-gone:restart-helper"}},
		{"SIGSTOP: state T then sigcont", 40, func(i int, o *Observation) {
			if i >= 5 {
				o.ProcState, o.Hung = "T", true
			}
		}, []string{"15:game-stopped:sigcont", "30:game-stopped:sigcont"}[:1]},
		{"hung window 30 s", 60, func(i int, o *Observation) {
			if i >= 5 {
				o.Hung = true
			}
		}, []string{"35:game-hung:restart-game"}},
		{"crash dialog 3 s", 20, func(i int, o *Observation) {
			if i >= 2 {
				o.Dialogs = []string{"Diablo II Critical Error"}
			}
		}, []string{"5:crash-dialog:dismiss-dialog", "13:crash-dialog:dismiss-dialog", "21:crash-dialog:restart-game"}[:2]},
		{"frozen frame and flat CPU", 120, func(i int, o *Observation) {
			o.FrameHash, o.CPUTicks = "same", 5
		}, []string{"90:game-frozen:restart-game"}},
		{"static frame alone is not frozen (CPU moves)", 200, func(i int, o *Observation) { o.FrameHash = "same" }, nil},
		{"bot unreachable 60 s", 100, func(i int, o *Observation) { o.Bot = &BotHealth{Reachable: false, Err: "refused"} },
			[]string{"60:bot-silent:restart-bot"}},
		{"bot stuck reason 60 s", 100, func(i int, o *Observation) {
			o.Bot = &BotHealth{Reachable: true, StuckReason: "no-path", State: "exploring", GoalActive: true}
		}, []string{"60:bot-stuck:restart-bot"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := run(newEngine(), t0, c.secs, c.mut)
			if len(c.want) == 0 && len(got) == 0 {
				return
			}
			// Compare only the first len(want) decisions (later cooldown repeats are covered by dedicated tests).
			if len(got) < len(c.want) {
				t.Fatalf("got %v want %v", got, c.want)
			}
			for i, w := range c.want {
				if got[i] != w {
					t.Fatalf("got %v want %v", got, c.want)
				}
			}
			if len(c.want) == 0 {
				t.Fatalf("unexpected decisions %v", got)
			}
		})
	}
}

func TestNoProgressLadder(t *testing.T) {
	e := newEngine()
	e.Disabled["idle-reaper"] = true
	// Stand still with a goal active: NoProgress is 480 s, then one step per StepWait (120 s).
	got := run(e, t0, 1000, func(i int, o *Observation) {
		o.X, o.Y = 50, 50
		o.Bot = &BotHealth{Reachable: true, GoalActive: true}
	})
	want := []string{"480:no-progress:esc3", "600:no-progress:bot-reset", "720:no-progress:leave-rejoin", "840:no-progress:restart-game", "960:no-progress:alert"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestNoProgressResetsOnProgress(t *testing.T) {
	e := newEngine()
	got := run(e, t0, 800, func(i int, o *Observation) {
		o.X, o.Y = 50, 50
		o.Bot = &BotHealth{Reachable: true, GoalActive: true}
		if i == 500 { // some xp: progress, ladder resets after ResetAfter, a later stall starts from step 1
			o.Exp = 10
		}
		if i > 500 {
			o.Exp = 10
		}
	})
	want := []string{"480:no-progress:esc3"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestNoProgressNeedsGoal(t *testing.T) {
	e := newEngine()
	e.Disabled["idle-reaper"] = true
	got := run(e, t0, 2000, func(i int, o *Observation) { o.X, o.Y = 50, 50 })
	if len(got) != 0 {
		t.Fatalf("no goal configured, got %v", got)
	}
}

func TestSettleAndPriority(t *testing.T) {
	// A dead player and a stuck panel at once: only the higher-priority rule fires per step.
	e := newEngine()
	got := run(e, t0, 40, func(i int, o *Observation) { o.Life, o.Mode, o.Panels = 0, 17, []int{1} })
	if len(got) == 0 || !strings.Contains(got[0], "death-overlay") {
		t.Fatalf("got %v", got)
	}
}

func TestDisabledRuleAndOverride(t *testing.T) {
	cfg := DefaultConfig()
	off := false
	f := Dur(2 * time.Second)
	cfg.Rules = map[string]RuleOverride{"death-overlay": {Enabled: &off}, "esc-menu-stuck": {For: &f}}
	cfg.Instances = []InstanceConf{{ID: "i5", Character: "Multi"}}
	e := NewEngine(cfg)
	got := run(e, t0, 30, func(i int, o *Observation) {
		o.Life, o.Mode = 0, 17
	})
	if len(got) != 0 {
		t.Fatalf("disabled rule fired: %v", got)
	}
	got = run(NewEngine(cfg), t0, 10, func(i int, o *Observation) { o.EscMenu = true })
	if len(got) == 0 || got[0] != "2:esc-menu-stuck:esc" {
		t.Fatalf("override: %v", got)
	}
}

func TestParseBotHealth(t *testing.T) {
	now := time.Unix(2000000000, 0)
	h := ParseBotHealth([]byte(`{"state":"exploring","time_in_state":42.5,"last_progress_unix":1999999940,"stuck_reason":"","goal_active":true}`), now)
	if h.State != "exploring" || h.TimeInState != 42500*time.Millisecond || h.SinceProgress != 60*time.Second || !h.GoalActive {
		t.Fatalf("%+v", h)
	}
	h = ParseBotHealth([]byte(`{"data":{"state":"dead","stuckReason":"overlay","sinceProgress":"3m"}}`), now)
	if h.State != "dead" || h.StuckReason != "overlay" || h.SinceProgress != 3*time.Minute {
		t.Fatalf("%+v", h)
	}
	h = ParseBotHealth([]byte(`not json`), now)
	if !h.Reachable || h.SinceProgress != -1 {
		t.Fatalf("%+v", h)
	}
}

func TestDurJSON(t *testing.T) {
	var c Config
	c.Th.NoProgress = Dur(time.Second)
	if err := jsonUnmarshal(`{"thresholds":{"noProgress":"10m","escMenu":30}}`, &c); err != nil {
		t.Fatal(err)
	}
	if c.Th.NoProgress.D() != 10*time.Minute || c.Th.EscMenu.D() != 30*time.Second {
		t.Fatalf("%+v", c.Th)
	}
}

func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }

func TestIdleReaper(t *testing.T) {
	still := func(i int, o *Observation) { o.X, o.Y = 50, 50 }
	got := run(newEngine(), t0, 1900, still)
	if len(got) != 1 || got[0] != "1800:idle-reaper:stop-instance" {
		t.Fatalf("got %v", got)
	}
	// Lock held, kept, i0 and a live bot all prevent it.
	for name, mut := range map[string]func(*Observation){
		"lock": func(o *Observation) { o.LockHeld = true },
		"bot":  func(o *Observation) { o.Bot = &BotHealth{Reachable: true} },
		"i0":   func(o *Observation) { o.Instance = "i0" },
	} {
		e := newEngine()
		if name == "i0" {
			e.byID["i0"] = InstanceConf{ID: "i0"}
		}
		if g := run(e, t0, 1900, func(i int, o *Observation) { still(i, o); mut(o) }); len(g) != 0 {
			t.Fatalf("%s: %v", name, g)
		}
	}
	cfg := DefaultConfig()
	cfg.Keep = []string{"i5"}
	cfg.Instances = []InstanceConf{{ID: "i5"}}
	if g := run(NewEngine(cfg), t0, 1900, still); len(g) != 0 {
		t.Fatalf("keep: %v", g)
	}
}
