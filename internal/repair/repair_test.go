package repair

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Unix(1_800_000_000, 0)

func open(allowed ...string) func(string) bool {
	return func(id string) bool {
		for _, a := range allowed {
			if a == id {
				return true
			}
		}
		return false
	}
}

func TestShippedTable(t *testing.T) {
	tb, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}
	if tb.Default != "manual" || len(tb.Modes) != 7 {
		t.Fatalf("table: %+v", tb)
	}
	for _, id := range []string{"manual", "break", "hour", "half", "always"} {
		if _, err := tb.Mode(id); err != nil {
			t.Error(err)
		}
	}
	if _, err := tb.Mode("sometimes"); err == nil || !strings.Contains(err.Error(), "sometimes") {
		t.Errorf("unknown mode error must name it: %v", err)
	}
}

func TestParseTableErrors(t *testing.T) {
	cases := map[string]string{
		"no test":     `{"warnPct":30,"default":"m","modes":[{"id":"m","label":"M"},{"id":"x","label":"X","unlock":"u"}]}`,
		"no unlock":   `{"warnPct":30,"default":"m","modes":[{"id":"m","label":"M"},{"id":"x","label":"X","always":true}]}`,
		"default act": `{"warnPct":30,"default":"m","modes":[{"id":"m","label":"M","always":true}]}`,
		"no default":  `{"warnPct":30,"default":"q","modes":[{"id":"m","label":"M"}]}`,
		"dup":         `{"warnPct":30,"default":"m","modes":[{"id":"m","label":"M"},{"id":"m","label":"M"}]}`,
		"warn":        `{"warnPct":0,"default":"m","modes":[{"id":"m","label":"M"}]}`,
	}
	for name, js := range cases {
		if _, err := ParseTable([]byte(js)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// Both classes: the controller is class-free, the gear differs (a caster's staff and a paladin's shield).
func gearFor(class string, durs map[string]int) func() []Cond {
	names := map[string]map[string]string{
		"sorceress": {"rhand": "Staff", "torso": "Robe", "head": "Cap"},
		"paladin":   {"rhand": "Sword", "lhand": "Shield", "torso": "Plate"},
	}[class]
	return func() []Cond {
		var out []Cond
		for slot, name := range names {
			if c, ok := Classify(slot, name, durs[slot], 100, 30); ok {
				out = append(out, c)
			}
		}
		return out
	}
}

func TestManualNeverRepairsOnItsOwn(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r, _ := New(Config{Gear: gearFor(class, map[string]int{"rhand": 0, "torso": 10}), Now: func() time.Time { return t0 }})
		if got := r.ModeID(); got != "manual" {
			t.Fatalf("%s: default mode %q", class, got)
		}
		if ok, req := r.Permit(Facts{Now: t0, Damaged: true, Broken: true}); ok || req {
			t.Errorf("%s: manual mode repaired without a request", class)
		}
	}
}

func TestRequestThenRepairedTwoClasses(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r, _ := New(Config{Gear: gearFor(class, map[string]int{"rhand": 50}), Now: func() time.Time { return t0 }})
		if _, err := r.Request(); err != nil {
			t.Fatal(err)
		}
		ok, req := r.Permit(Facts{Now: t0, Damaged: true})
		if !ok || !req {
			t.Fatalf("%s: request not honoured", class)
		}
		if r.View().Request.State != StateRequested {
			t.Fatal("state not requested")
		}
		r.Repaired(t0.Add(time.Minute))
		v := r.View()
		if v.Request.State != StateDone || v.Request.Done == 0 {
			t.Fatalf("state after repair: %+v", v.Request)
		}
		if ok, _ := r.Permit(Facts{Now: t0, Damaged: true}); ok {
			t.Error("manual mode repaired again after the request was done")
		}
	}
}

func TestRequestRefusedWhenUndamaged(t *testing.T) {
	r, _ := New(Config{Gear: gearFor("paladin", map[string]int{"rhand": 100, "lhand": 100, "torso": 100})})
	_, err := r.Request()
	var ref *Refusal
	if !errors.As(err, &ref) || ref.Code != "undamaged" {
		t.Fatalf("got %v", err)
	}
}

func TestBlockedKeepsRequestAndCancel(t *testing.T) {
	r, _ := New(Config{Gear: gearFor("sorceress", map[string]int{"rhand": 50}), Now: func() time.Time { return t0 }})
	r.Request()
	r.Blocked("not enough gold")
	if v := r.View(); v.Request.State != StateRequested || v.Request.Note != "not enough gold" {
		t.Fatalf("%+v", v.Request)
	}
	if err := r.Cancel(); err != nil {
		t.Fatal(err)
	}
	if q := r.View().Request; q == nil || q.State != StateCancel || r.Requested() {
		t.Errorf("cancel must close the request and keep it as cancelled: %+v", q)
	}
	if err := r.Cancel(); err == nil {
		t.Error("second cancel must be refused")
	}
}

func TestModes(t *testing.T) {
	mode := "manual"
	r, _ := New(Config{Mode: func() string { return mode }, Allowed: open("repair_break", "repair_hour", "repair_always")})
	f := func(now time.Time, dmg, brk bool) Facts { return Facts{Now: now, Damaged: dmg, Broken: brk} }

	mode = "break"
	if ok, _ := r.Permit(f(t0, true, false)); ok {
		t.Error("break mode repaired unbroken gear")
	}
	if ok, req := r.Permit(f(t0, true, true)); !ok || req {
		t.Error("break mode ignored broken gear")
	}
	mode = "hour"
	if ok, _ := r.Permit(f(t0, true, false)); !ok {
		t.Error("hour: first repair is due at once")
	}
	r.Repaired(t0)
	if ok, _ := r.Permit(f(t0.Add(59*time.Minute), true, false)); ok {
		t.Error("hour: due too early")
	}
	if ok, _ := r.Permit(f(t0.Add(61*time.Minute), true, false)); !ok {
		t.Error("hour: due after an hour")
	}
	if ok, _ := r.Permit(f(t0.Add(2*time.Hour), false, false)); ok {
		t.Error("nothing damaged must never repair")
	}
	mode = "always"
	if ok, _ := r.Permit(f(t0, true, false)); !ok {
		t.Error("always: damaged gear not repaired")
	}
}

func TestLockedModeActsAsManual(t *testing.T) {
	mode := "half" // not bought
	r, _ := New(Config{Mode: func() string { return mode }, Allowed: open()})
	if r.ModeID() != "manual" {
		t.Fatalf("locked mode in force: %s", r.ModeID())
	}
	if ok, _ := r.Permit(Facts{Now: t0, Damaged: true}); ok {
		t.Error("locked mode repaired")
	}
	mode = "nonsense"
	if r.ModeID() != "manual" {
		t.Error("unknown mode must act as manual")
	}
	v := r.View()
	locked := 0
	for _, m := range v.Modes {
		if m.Locked {
			locked++
		}
	}
	if locked != 4 {
		t.Errorf("locked modes %d, want 4", locked)
	}
}

func TestClassifyAndWorst(t *testing.T) {
	a, _ := Classify("head", "Cap", 5, 20, 30)
	b, _ := Classify("torso", "Robe", 0, 40, 30)
	c, _ := Classify("rhand", "Staff", 90, 100, 30)
	if !a.Warn || a.Broken || a.Pct != 25 || !b.Broken || b.Warn || c.Warn {
		t.Fatalf("%+v %+v %+v", a, b, c)
	}
	if _, ok := Classify("rring", "Ring", 0, 0, 30); ok {
		t.Error("an item without durability has no condition")
	}
	if w := Worst([]Cond{a, c}); w != "Low durability: Cap 25%" {
		t.Error(w)
	}
	if w := Worst([]Cond{a, b, c}); w != "Broken: Robe" {
		t.Error(w)
	}
	if Worst([]Cond{c}) != "" {
		t.Error("healthy gear raised a line")
	}
}

func TestPersistence(t *testing.T) {
	p := filepath.Join(t.TempDir(), "r.json")
	g := gearFor("paladin", map[string]int{"rhand": 10})
	r, _ := New(Config{Path: p, Gear: g, Now: func() time.Time { return t0 }})
	r.Request()
	r2, err := New(Config{Path: p})
	if err != nil {
		t.Fatal(err)
	}
	if ok, req := r2.Permit(Facts{Now: t0, Damaged: true}); !ok || !req {
		t.Error("request lost across a restart")
	}
}

// A Pact sets the hidden modes (data/pacts/pacts.json): never refuses even a request, low repairs under its percent.
func TestPactModes(t *testing.T) {
	mode := "never"
	r, _ := New(Config{Mode: func() string { return mode }, Allowed: open()})
	if r.ModeID() != "never" {
		t.Fatalf("hidden mode needs no unlock: %s", r.ModeID())
	}
	var ref *Refusal
	if _, err := r.Request(); !errors.As(err, &ref) || ref.Code != "pact" {
		t.Errorf("request under never: %v", err)
	}
	if ok, _ := r.Permit(Facts{Now: t0, Damaged: true, Broken: true, Lowest: 0}); ok {
		t.Error("never repaired")
	}
	mode = "low"
	if ok, _ := r.Permit(Facts{Now: t0, Damaged: true, Lowest: 25}); ok {
		t.Error("low repaired at 25")
	}
	if ok, _ := r.Permit(Facts{Now: t0, Damaged: true, Lowest: 19}); !ok {
		t.Error("low did not repair at 19")
	}
	for _, m := range r.View().Modes {
		if m.ID == "never" || m.ID == "hour" && m.Locked == false {
			t.Errorf("view lists %+v", m)
		}
	}
}

func TestPendingRequestDoesNotBeatNever(t *testing.T) {
	mode := "manual"
	r, _ := New(Config{Mode: func() string { return mode }, Now: func() time.Time { return t0 }})
	if _, err := r.Request(); err != nil {
		t.Fatal(err)
	}
	mode = "never" // a Pact starts after the request
	if ok, _ := r.Permit(Facts{Now: t0, Damaged: true}); ok {
		t.Error("a pending request overrode the Pact")
	}
}

func TestShortKeepsRequestOpenWithAmount(t *testing.T) {
	r, _ := New(Config{Gear: gearFor("paladin", map[string]int{"rhand": 0}), Now: func() time.Time { return t0 }})
	r.Request()
	r.Short(41, 116)
	v := r.View().Request
	if v.State != StateRequested || v.Short != 41 || v.Need != 116 || !strings.Contains(v.Note, "need 41 more") {
		t.Fatalf("%+v", v)
	}
	if !r.Requested() {
		t.Error("request closed")
	}
	r.Repaired(t0)
	if v := r.View().Request; v.State != StateDone || v.Short != 0 {
		t.Errorf("after repair %+v", v)
	}
}
