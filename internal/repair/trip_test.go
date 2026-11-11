package repair

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type rig struct {
	r    *Controller
	now  time.Time
	gold int
	est  int
	mode string
}

func newRig(t *testing.T, class string, path string) *rig {
	t.Helper()
	g := &rig{now: t0, gold: 100, est: 200, mode: "manual"}
	r, err := New(Config{
		Path: path, Gear: gearFor(class, worn),
		Estimate: func() int { return g.est }, Gold: func() int { return g.gold },
		Mode: func() string { return g.mode }, Now: func() time.Time { return g.now },
	})
	if err != nil {
		t.Fatal(err)
	}
	g.r = r
	return g
}

var classes = []string{"sorceress", "paladin"}

var (
	worn = map[string]int{"rhand": 40, "torso": 70, "head": 80, "lhand": 90}
	full = map[string]int{"rhand": 100, "torso": 100, "head": 100, "lhand": 100}
)

// The button reads from live pack gold against the estimate x 1.3 (a lower bound).
func TestButtonStateFromLiveGold(t *testing.T) {
	for _, class := range classes {
		g := newRig(t, class, "")
		v := g.r.View()
		if v.Schema != 2 || v.Trip == nil {
			t.Fatalf("%s: schema %d trip %v", class, v.Schema, v.Trip)
		}
		// need = ceil(200 x 1.3) = 260; gold 100
		if v.Trip.State != StateFarmable || v.Trip.Button.Label != "Farm until repair" || v.Trip.Need != 260 {
			t.Fatalf("%s: %+v", class, v.Trip)
		}
		if want := "need 160 more gold (have 100, estimate 200)"; v.Trip.Line != want {
			t.Errorf("%s: line %q want %q", class, v.Trip.Line, want)
		}
		g.gold = 259
		if g.r.View().Trip.State != StateFarmable {
			t.Errorf("%s: 259 gold is under 260", class)
		}
		g.gold = 260
		v = g.r.View()
		if v.Trip.State != StateReady || v.Trip.Button.Label != "Repair now" || !v.Trip.Button.Enabled || v.Trip.Line != "" {
			t.Errorf("%s: %+v", class, v.Trip)
		}
	}
}

func TestButtonDisabledWithReason(t *testing.T) {
	g := newRig(t, "paladin", "")
	g.r.c.Gear = gearFor("paladin", full)
	v := g.r.View().Trip
	if v.State != StateDisabled || v.Button.Enabled || v.Button.Reason != "nothing worn is damaged" {
		t.Errorf("undamaged: %+v", v)
	}
	var ref *Refusal
	if _, err := g.r.Request(); !errors.As(err, &ref) || ref.Code != "undamaged" {
		t.Errorf("undamaged request: %v", err)
	}
	// a Pact's never mode
	g2 := newRig(t, "sorceress", "")
	g2.mode = "never"
	if v := g2.r.View().Trip; v.State != StateDisabled || !strings.Contains(v.Button.Reason, "Pact") {
		t.Errorf("never: %+v", v)
	}
	if _, err := g2.r.Request(); !errors.As(err, &ref) || ref.Code != "pact" {
		t.Errorf("never request: %v", err)
	}
}

func TestRequestKindFollowsGoldAndOneAtATime(t *testing.T) {
	for _, class := range classes {
		g := newRig(t, class, "")
		q, err := g.r.Request()
		if err != nil || q.Kind != KindFarm || q.Phase != PhaseFarming || q.Target != 260 || q.FarmLegs != 1 {
			t.Fatalf("%s: %+v %v", class, q, err)
		}
		var ref *Refusal
		if _, err := g.r.Request(); !errors.As(err, &ref) || ref.Code != "active" {
			t.Errorf("%s: second request: %v", class, err)
		}
		if v := g.r.View().Trip; v.Button.Action != "cancel" || v.Button.Label != "Cancel" || v.State != PhaseFarming {
			t.Errorf("%s: %+v", class, v)
		}
		g.r.FarmProgress(140, 300, "Stony Field")
		if got := g.r.View().Trip.Line; got != "farming for repair: 140 of 300" {
			t.Errorf("progress line %q", got)
		}
		g.gold = 300
		g2 := newRig(t, class, "")
		g2.gold = 300
		if q, _ := g2.r.Request(); q.Kind != KindRepair || q.Phase != PhaseQueued {
			t.Errorf("%s: enough gold: %+v", class, q)
		}
	}
}

// Farm then repair then return: the whole life of a request, with durability before and after.
func TestFarmThenTripThenDone(t *testing.T) {
	for _, class := range classes {
		g := newRig(t, class, "")
		g.r.Request()
		g.r.FarmReached()
		q, _ := g.r.Active()
		if q.Kind != KindRepair || q.Phase != PhaseQueued {
			t.Fatalf("%s after the farm leg: %+v", class, q)
		}
		if err := g.r.BeginTrip(g.now); err != nil {
			t.Fatal(err)
		}
		g.r.SetPhase(PhaseRepairing, "")
		g.r.Repaired(g.now) // the town paid the smith
		if q, _ := g.r.Active(); q.Phase != PhaseReturning {
			t.Fatalf("%s: %+v", class, q)
		}
		g.r.c.Gear = gearFor(class, full)
		g.r.EndTrip(g.now, true, false, "")
		v := g.r.View()
		if g.r.Requested() || v.Request.State != StateDone || v.Request.DurBefore != 40 || v.Request.DurAfter != 100 {
			t.Fatalf("%s: %+v", class, v.Request)
		}
		if !strings.Contains(v.Trip.Line, "durability 40% to 100%") || v.Trip.State != StateDone {
			t.Errorf("%s: %+v", class, v.Trip)
		}
	}
}

// Guards: the cooldown between trips, the trip cap, the failed-request cooldown, auto cancel after a farm leg.
func TestLoopGuards(t *testing.T) {
	for _, class := range classes {
		g := newRig(t, class, "")
		g.gold = 300
		g.r.Request()
		if err := g.r.BeginTrip(g.now); err != nil {
			t.Fatal(err)
		}
		g.r.EndTrip(g.now, false, false, "the smith could not be reached")
		q, _ := g.r.Active()
		if q.Phase != PhaseQueued || q.Trips != 1 {
			t.Fatalf("%s: %+v", class, q)
		}
		var ref *Refusal
		if err := g.r.BeginTrip(g.now.Add(5 * time.Minute)); !errors.As(err, &ref) || ref.Code != "cooldown" {
			t.Fatalf("%s: second trip inside the cooldown: %v", class, err)
		}
		g.now = g.now.Add(21 * time.Minute)
		if err := g.r.BeginTrip(g.now); err != nil {
			t.Fatalf("%s: after the cooldown: %v", class, err)
		}
		g.r.EndTrip(g.now, false, false, "menu failed")
		if g.r.Requested() || g.r.View().Request.State != StateFailed {
			t.Fatalf("%s: two trips must fail the request: %+v", class, g.r.View().Request)
		}
		// a failed request starts the cooldown for the next one
		if v := g.r.View().Trip; v.Button.Enabled || !strings.Contains(v.Button.Reason, "try again in") {
			t.Errorf("%s: %+v", class, v)
		}
		if _, err := g.r.Request(); !errors.As(err, &ref) || ref.Code != "cooldown" {
			t.Errorf("%s: request in the cooldown: %v", class, err)
		}
		g.now = g.now.Add(21 * time.Minute)
		if _, err := g.r.Request(); err != nil {
			t.Errorf("%s: after the cooldown: %v", class, err)
		}
	}
}

func TestUnpayableAfterFarmLegCancels(t *testing.T) {
	g := newRig(t, "paladin", "")
	g.r.Request() // farm kind, leg 1
	g.r.FarmReached()
	g.r.BeginTrip(g.now)
	g.r.EndTrip(g.now, false, true, "need 300, have 120")
	q := g.r.View().Request
	if q.State != StateCancel || !q.Auto || !strings.Contains(q.Note, "still short of gold after farming") {
		t.Fatalf("%+v", q)
	}
	// a request that started as a repair and met a higher price gets one farm leg first
	g2 := newRig(t, "sorceress", "")
	g2.gold = 300
	g2.r.Request()
	g2.r.BeginTrip(g2.now)
	g2.r.EndTrip(g2.now, false, true, "need 400, have 300")
	if q, _ := g2.r.Active(); q.Kind != KindFarm || q.Phase != PhaseFarming || q.FarmLegs != 1 {
		t.Fatalf("%+v", q)
	}
	g2.r.FarmReached()
	g2.r.BeginTrip(g2.now.Add(25 * time.Minute))
	g2.r.EndTrip(g2.now, false, true, "need 400, have 310")
	if q := g2.r.View().Request; q.State != StateCancel || !q.Auto {
		t.Fatalf("second short must cancel: %+v", q)
	}
}

func TestPlayerCancelHasNoCooldown(t *testing.T) {
	g := newRig(t, "sorceress", "")
	g.r.Request()
	if err := g.r.Cancel(); err != nil {
		t.Fatal(err)
	}
	if v := g.r.View().Trip; v.State != StateCancel || !v.Button.Enabled {
		t.Errorf("%+v", v)
	}
	if _, err := g.r.Request(); err != nil {
		t.Errorf("a taken-back click must not lock the button: %v", err)
	}
}

// The request survives a restart; a trip that was under way starts over, a paid repair is done.
func TestTripPersistsAndRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), StateFile)
	g := newRig(t, "paladin", path)
	g.gold = 300
	g.r.Request()
	g.r.BeginTrip(g.now)
	g.r.SetPhase(PhaseRepairing, "")
	g2 := newRig(t, "paladin", path)
	q, ok := g2.r.Active()
	if !ok || q.Phase != PhaseQueued || q.Trips != 1 || q.Kind != KindRepair {
		t.Fatalf("restart mid-trip: %+v %v", q, ok)
	}
	g2.r.BeginTrip(g2.now.Add(30 * time.Minute))
	g2.r.SetPhase(PhaseRepairing, "")
	g2.r.Repaired(g2.now)
	g3 := newRig(t, "paladin", path)
	if g3.r.Requested() || g3.r.View().Request.State != StateDone {
		t.Errorf("restart after the repair: %+v", g3.r.View().Request)
	}
	// farming stays farming
	path2 := filepath.Join(t.TempDir(), StateFile)
	f := newRig(t, "sorceress", path2)
	f.r.Request()
	f.r.FarmProgress(140, 300, "Stony Field")
	f2 := newRig(t, "sorceress", path2)
	if q, _ := f2.r.Active(); q.Phase != PhaseFarming || q.Have != 140 || q.Target != 300 {
		t.Errorf("%+v", q)
	}
}

// An old state file (before the trip) loads and its pending request is queued.
func TestOldStateFileLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), StateFile)
	if err := writeFile(path, `{"request":{"state":"requested","at":1800000000},"lastRepair":5}`); err != nil {
		t.Fatal(err)
	}
	g := newRig(t, "sorceress", path)
	v := g.r.View()
	if v.Request == nil || v.Trip.State != PhaseQueued || v.Trip.Button.Action != "cancel" {
		t.Errorf("%+v %+v", v.Request, v.Trip)
	}
}

func TestTripKnobsAreData(t *testing.T) {
	tb, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}
	if tb.Trip.GoldMult != 1.3 || tb.Trip.CooldownSecs != 1200 || tb.Trip.MaxTrips != 2 || tb.Trip.SafeWaitSecs != 60 || tb.Trip.PortalClearTiles != 10 {
		t.Errorf("%+v", tb.Trip)
	}
	if _, err := ParseTable([]byte(`{"warnPct":30,"default":"m","trip":{"goldMult":0.5},"modes":[{"id":"m","label":"M"}]}`)); err == nil {
		t.Error("a multiplier under 1 must be refused")
	}
	if tb, err := ParseTable([]byte(`{"warnPct":30,"default":"m","modes":[{"id":"m","label":"M"}]}`)); err != nil || tb.Trip.MaxTrips != 2 {
		t.Errorf("missing trip block takes the defaults: %v %+v", err, tb)
	}
}

func writeFile(p, s string) error { return os.WriteFile(p, []byte(s), 0o644) }
