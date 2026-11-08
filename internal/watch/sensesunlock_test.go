package watch

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/senses"
	"d2lod-bot/internal/unlock"
)

type snapSrc struct{ s senses.Snapshot }

func (f snapSrc) Snapshot() (senses.Snapshot, bool) { return f.s, true }

func sensesSrc(t *testing.T) snapSrc {
	t.Helper()
	snap, ok := sensesFixture(t).Snapshot()
	if !ok {
		t.Fatal("no snapshot")
	}
	snap.Unit = []senses.Unit{{ID: 1, Kind: senses.KindMonster, X: 3, Y: 1, Class: 5, Name: "Fallen", LifePct: 80, Target: true}}
	return snapSrc{snap}
}

func shippedEngine(t *testing.T) *unlock.Engine {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	e, err := unlock.Open(t.TempDir(), "sorceress", l, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

var tierSteps = []unlock.Event{
	{},                           // tier 0
	{Type: unlock.EvLevel, N: 2}, // tier 1 (alt trigger: clvl 2)
	{Type: unlock.EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"},         // tiers 2 and 3
	{Type: unlock.EvCodexFirst, Sub: "champion", ID: "champion_pack", Name: "Cursed"}, // tier 4
}

// TestSensesLadderGolden is the gating golden for tiers 0 to 4: what each state of the ladder sends and what stays absent.
func TestSensesLadderGolden(t *testing.T) {
	eng := shippedEngine(t)
	s := &Senses{Src: sensesSrc(t), Lock: LadderUnlocker{Engine: eng}}
	for step, ev := range tierSteps {
		if ev.Type != "" {
			if _, err := eng.Observe(ev); err != nil {
				t.Fatal(err)
			}
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
		golden(t, "senses-ladder-step"+string(rune('0'+step)), w.Body.String())
		body := w.Body.String()
		for _, k := range []string{`"map"`, `"hero"`, `"units"`} {
			want := (k != `"units"` && step >= 1) || (k == `"units"` && step >= 2)
			if strings.Contains(body, k) != want {
				t.Errorf("step %d: key %s present=%v", step, k, !want)
			}
		}
	}
}

func TestSensesLadderLockedListAndNext(t *testing.T) {
	eng := shippedEngine(t)
	s := &Senses{Src: sensesSrc(t), Lock: LadderUnlocker{Engine: eng}}
	r := s.Reply(0)
	if len(r.Tiers) != 0 || len(r.Locked) != 7 || r.Next == nil || r.Next.Tier != 1 || r.Next.Title != "Minimap" {
		t.Fatalf("%+v", r)
	}
	for _, l := range r.Locked {
		if l.Teaser == "" || strings.Contains(l.Teaser, "New sense") {
			t.Errorf("locked %+v: no teaser or a spoiler", l)
		}
	}
	eng.Observe(unlock.Event{Type: unlock.EvLevel, N: 2})
	r = s.Reply(0)
	if len(r.Tiers) != 1 || len(r.Locked) != 6 || r.Next.Tier != 2 {
		t.Fatalf("%+v", r)
	}
}

func TestSensesLadderCapIsAMaximum(t *testing.T) {
	eng := shippedEngine(t)
	eng.Observe(unlock.Event{Type: unlock.EvLevel, N: 2})
	eng.Observe(unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"})
	free := LadderUnlocker{Engine: eng}
	capped := LadderUnlocker{Engine: eng, Cap: 1}
	if !free.Allowed(2) || capped.Allowed(2) || !capped.Allowed(1) {
		t.Fatal("cap")
	}
	if got := (&Senses{Src: sensesSrc(t), Lock: capped}).Reply(0); len(got.Tiers) != 1 || got.Units != nil || len(got.Locked) != 0 {
		t.Fatalf("%+v", got)
	}
	if !free.Met("Fallen") || free.Met("Griswold") || capped.Met("Fallen") {
		t.Fatal("per-class part of tier 3")
	}
	if (LadderUnlocker{}).Allowed(1) {
		t.Fatal("no engine opened a tier")
	}
}

func TestSensesLadderTriggers(t *testing.T) {
	cases := []struct {
		name string
		ev   unlock.Event
		tier int
	}{
		{"area first", unlock.Event{Type: unlock.EvCodexFirst, Sub: "area", ID: "blood_moor", Name: "Blood Moor"}, 1},
		{"superunique", unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "corpsefire", Name: "Corpsefire"}, 4},
		{"first drop", unlock.Event{Type: unlock.EvCodexFirst, Sub: "drop", ID: "first_unique", Name: "unique"}, 5},
		{"act boss", unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "andariel", Name: "Andariel"}, 7},
		{"clvl 12", unlock.Event{Type: unlock.EvLevel, N: 12}, 7},
	}
	for _, c := range cases {
		eng := shippedEngine(t)
		if _, err := eng.Observe(c.ev); err != nil || !eng.SenseTier(c.tier) {
			t.Errorf("%s: tier %d not open (%v)", c.name, c.tier, err)
		}
	}
	eng := shippedEngine(t)
	eng.Observe(unlock.Event{Type: unlock.EvLevel, N: 11})
	if eng.SenseTier(7) || eng.SenseTier(6) {
		t.Fatal("tiers 6 and 7 opened early; 6 is bought only")
	}
}

// Tiers 5 and 6 are bought: refused short, accepted with the ledger entry, then the senses reply lists them.
func TestSensesLadderBuyTiers(t *testing.T) {
	r, eng := unlockRigShipped(t, "Sorceress")
	for _, ev := range funStream() {
		if ev.Type != goals.EvDrop { // the first unique drop would open tier 5 for free
			r.fun.OnGoalEvent(ev)
		}
	}
	r.fun.Settle()
	before, _ := r.ledger.Balance("tally")
	post := func(id string) int {
		resp, _ := r.do("POST", r.answer, UnlocksBuyRoute, funToken, `{"id":"`+id+`"}`)
		return resp.StatusCode
	}
	if before.Spendable < 40 || before.Spendable >= 80 {
		t.Fatalf("fixture balance %d must sit between the two prices", before.Spendable)
	}
	if post("senses_danger") != http.StatusPaymentRequired || eng.SenseTier(6) {
		t.Fatal("short balance must be refused")
	}
	if post("senses_loot") != 200 || !eng.SenseTier(5) { // 40
		t.Fatal("tier 5 not bought")
	}
	if post("senses_loot") != http.StatusConflict {
		t.Fatal("second buy")
	}
	bal, _ := r.ledger.Balance("tally")
	if bal.Spendable != before.Spendable-40 {
		t.Fatalf("balance %d", bal.Spendable)
	}
}

func unlockRigShipped(t *testing.T, class string) (*funRig, *unlock.Engine) {
	t.Helper()
	eng := shippedEngine(t)
	r := newFunRigWith(t, class, 1, func(_ *funRig, fc *FunConfig) { fc.Unlock = eng })
	return r, eng
}

// The whole path for two classes: the Codex firsts of a run open the same tiers (monster and area firsts are world data).
func TestSensesLadderThroughFunTwoClasses(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		r, eng := unlockRigShipped(t, class)
		r.feed(funStream()...)
		for tier, want := range map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: false, 7: false} {
			// tier 5 opens by the first unique drop of the stream (its alt trigger)
			if eng.SenseTier(tier) != want {
				t.Errorf("%s tier %d open=%v, want %v", class, tier, !want, want)
			}
		}
		if got := storyLines(r); len(got) < 5 || !strings.Contains(strings.Join(got, "|"), "New sense") {
			t.Errorf("%s reveals: %v", class, got)
		}
	}
}

func TestNewSensesLock(t *testing.T) {
	eng := shippedEngine(t)
	eng.Observe(unlock.Event{Type: unlock.EvLevel, N: 2})
	eng.Observe(unlock.Event{Type: unlock.EvCodexFirst, Sub: "monster", ID: "fallen", Name: "Fallen"})
	if l := NewSensesLock(nil, 1); l.Allowed(2) || !l.Allowed(1) {
		t.Fatal("no ladder: the knob is a fixed tier")
	}
	if l := NewSensesLock(eng, 1); !l.Allowed(2) || !l.Allowed(3) {
		t.Fatal("the default knob 1 must not cap the ladder")
	}
	if l := NewSensesLock(eng, 2); !l.Allowed(2) || l.Allowed(3) {
		t.Fatal("a knob above 1 is a maximum")
	}
}
