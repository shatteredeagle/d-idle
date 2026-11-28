package arbiter

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata golden files")

const (
	wood   = "dark_wood"
	plains = "cold_plains"
	tick   = 10.0
	needG  = 300
)

// world is the tiny fake: three areas and a few facts.
type world struct {
	now, questAcc            float64
	area                     string
	progress                 uint64
	belt, packGold, stash    int
	quest                    int // 0 open, 1 done
	kills                    int
	life                     float64
	order                    float64 // seconds of player order left
	shopEmpty                bool
	changes                  int
	handedIn, choreWithdrawn bool
}

func newWorld() *world { return &world{area: Town, belt: 2, packGold: 500, life: 1, shopEmpty: true} }

func (w *world) view() View {
	return View{Now: w.now, Area: w.area, Progress: w.progress, Facts: map[string]int{
		"belt": w.belt, "pack_gold": w.packGold, "stash_gold": w.stash, "life_pct": int(w.life * 100),
	}}
}

func (w *world) sources() []Source {
	return []Source{
		FuncSource{N: "survive", F: func(View) *Intent {
			if w.life < 0.3 {
				return &Intent{Kind: "drink", Class: Reflex, Urgency: 1, NeedsArea: Any, EstCost: 2, Reason: "life under 30%"}
			}
			return nil
		}},
		FuncSource{N: "player", F: func(View) *Intent {
			if w.order > 0 {
				return &Intent{Kind: "order", Class: Player, Urgency: 1, NeedsArea: Any, EstCost: w.order, Reason: "player order", Resumable: true}
			}
			return nil
		}},
		FuncSource{N: "resupply", F: func(v View) *Intent {
			if w.belt >= 4 {
				return nil
			}
			in := Intent{Kind: "refill", Class: Upkeep, Urgency: 0.5, NeedsArea: Town, EstCost: 60, Reason: fmt.Sprintf("belt %d under 4", w.belt)}
			liquid := w.packGold + w.stash
			switch {
			case liquid < needG:
				return nil // cannot pay at all: recovery's business
			case w.packGold < needG:
				in.Steps = []string{fmt.Sprintf("withdraw_gold %d", needG-w.packGold), "buy_potions"}
				in.Reason += fmt.Sprintf(", pack %d + stash %d covers %d", w.packGold, w.stash, needG)
			default:
				in.Steps = []string{"buy_potions"}
			}
			return &in
		}},
		FuncSource{N: "recovery", F: func(View) *Intent {
			if w.belt < 4 && w.packGold+w.stash < needG {
				return &Intent{Kind: "farm_gold", Class: Upkeep, Urgency: 0.4, NeedsArea: plains, EstCost: 600, Reason: "broke: liquid gold under 300"}
			}
			return nil
		}},
		FuncSource{N: "quest", F: func(View) *Intent {
			if w.quest == 0 {
				return &Intent{Kind: "step", Class: Objective, Urgency: 0.6, NeedsArea: wood, EstCost: 120, Reason: "search_for_cain: tree", Resumable: true}
			}
			return nil
		}, Town: func(View) *Intent {
			if w.quest == 1 && !w.handedIn {
				return &Intent{Kind: "hand_in", Class: Objective, NeedsArea: Town, EstCost: 15, Reason: "report to Akara"}
			}
			return nil
		}},
		FuncSource{N: "farm", F: func(View) *Intent {
			return &Intent{Kind: "farm", Class: Filler, Urgency: 0.3, NeedsArea: plains, EstCost: 300, Reason: "level in cold plains"}
		}},
		FuncSource{N: "idle", F: func(View) *Intent {
			return &Intent{Kind: "idle", Class: Filler, Urgency: 0.1, NeedsArea: Any, Reason: "nothing to do"}
		}},
	}
}

// exec applies one chosen intent for one tick. report is nil for the baseline.
func (w *world) exec(in *Intent, report func(Intent, float64, bool)) {
	if in == nil {
		w.now += tick
		return
	}
	if in.NeedsArea != Any && in.NeedsArea != w.area {
		w.area = in.NeedsArea
		w.changes++
		w.now += 20
		return
	}
	ok := true
	switch in.Kind {
	case "step":
		if w.questAcc += tick; w.questAcc >= 120 {
			w.quest = 1
			w.progress++
		}
	case "refill":
		ok = !w.shopEmpty
		if ok {
			w.belt = 4
			w.progress++
		}
	case "farm", "farm_gold":
		w.kills++
		w.progress++
	case "drink":
		w.life = 1
	case "order":
		w.order -= tick
	}
	if report != nil {
		report(*in, w.now, ok)
	}
	w.now += tick
}

// naive is the baseline: the first-match if-chain of runGoal, reduced.
func naive(w *world) *Intent {
	switch {
	case w.belt < 4 && w.area != Town:
		return &Intent{Kind: "refill", NeedsArea: Town}
	case w.quest == 0 && w.area != wood:
		return &Intent{Kind: "step", NeedsArea: wood}
	case w.quest == 0:
		return &Intent{Kind: "step", NeedsArea: wood}
	}
	return &Intent{Kind: "farm", NeedsArea: plains}
}

func TestPingPong(t *testing.T) {
	base := newWorld()
	for base.now < 3600 {
		base.exec(naive(base), nil)
	}
	if base.changes <= 20 || base.quest != 0 || base.progress != 0 {
		t.Fatalf("baseline should flail: changes %d quest %d progress %d", base.changes, base.quest, base.progress)
	}

	w := newWorld()
	a := New(DefaultConfig(), w.sources()...)
	var questDoneAt float64
	var refused bool
	for w.now < 3600 {
		d := a.Decide(w.view())
		for _, r := range d.Rejected {
			if strings.Contains(r.Why, "travel budget") && strings.Contains(r.Why, "belt 2 under 4") {
				refused = true
			}
		}
		w.exec(d.Chosen, a.Report)
		if w.quest == 1 && questDoneAt == 0 {
			questDoneAt = w.now
		}
	}
	t.Logf("baseline: %d area changes, 0 progress; arbiter: %d area changes, quest done at %.0fs, kills %d", base.changes, w.changes, questDoneAt, w.kills)
	if w.changes > 12 {
		t.Errorf("arbiter changed area %d times in an hour, want <= 12", w.changes)
	}
	if questDoneAt == 0 || questDoneAt > 300 {
		t.Errorf("quest step did not get its dwell: done at %v", questDoneAt)
	}
	if !refused {
		t.Error("the refusal of the undo trip was never logged with both reasons")
	}
}

func TestBrokeWithStashGold(t *testing.T) {
	w := newWorld()
	w.packGold, w.stash = 20, 128000
	a := New(DefaultConfig(), w.sources()...)
	d := a.Decide(w.view())
	c := d.Chosen
	if c == nil || c.Source != "resupply" || len(c.Steps) != 2 || c.Steps[0] != "withdraw_gold 280" {
		t.Fatalf("want resupply with a withdraw step, got %v", d)
	}
	for _, r := range d.Rejected {
		if r.Intent.Source == "recovery" {
			t.Errorf("recovery proposed with %d gold in the stash: %v", w.stash, d)
		}
	}
	// batching: the quest source's hand-in rides the same town visit
	w.quest = 1
	if d = a.Decide(w.view()); len(d.Batch) != 1 || d.Batch[0].Key() != "quest/hand_in" {
		t.Errorf("want the hand-in batched into the town visit, got %v", d)
	}
	// really broke: now recovery is the right answer
	w.stash = 0
	d = New(DefaultConfig(), w.sources()...).Decide(w.view())
	if d.Chosen == nil || d.Chosen.Source != "recovery" {
		t.Errorf("want recovery when liquid gold is short, got %v", d)
	}
}

func TestReflexPreemptsAtOnce(t *testing.T) {
	w := newWorld()
	w.belt = 4
	a := New(DefaultConfig(), w.sources()...)
	a.Decide(w.view()) // quest takes the lease, inside its dwell
	w.area = wood
	d := a.Decide(w.view())
	if d.Chosen.Source != "quest" {
		t.Fatalf("setup: %v", d)
	}
	w.now += tick
	w.life = 0.2
	d = a.Decide(w.view())
	if d.Chosen.Source != "survive" || d.Chosen.Class != Reflex {
		t.Fatalf("reflex must preempt inside a lease: %v", d)
	}
	w.exec(d.Chosen, a.Report)
	if d = a.Decide(w.view()); d.Chosen.Source != "quest" || !d.Resumed {
		t.Errorf("quest should resume after the reflex: %v", d)
	}
}

func TestPlayerOrder(t *testing.T) {
	w := newWorld()
	w.belt, w.area = 4, wood
	a := New(DefaultConfig(), w.sources()...)
	a.Decide(w.view())
	w.now += tick
	w.order = 100
	d := a.Decide(w.view())
	if d.Chosen.Source != "player" {
		t.Fatalf("player order must preempt an objective lease: %v", d)
	}
	w.now += tick
	w.life = 0.1
	d = a.Decide(w.view())
	if d.Chosen.Source != "survive" {
		t.Fatalf("reflex must preempt the player order: %v", d)
	}
	w.exec(d.Chosen, a.Report)
	d = a.Decide(w.view())
	if d.Chosen.Source != "player" || !d.Resumed {
		t.Fatalf("player order should resume after the reflex: %v", d)
	}
	w.order = 0
	if d = a.Decide(w.view()); d.Chosen.Source != "quest" {
		t.Errorf("objective follows when the order ends: %v", d)
	}
}

func TestStarvation(t *testing.T) {
	w := newWorld()
	w.belt, w.area = 4, plains
	// upkeep appears while a filler lease runs: it wins at once
	a := New(DefaultConfig(), FuncSource{N: "farm", F: func(View) *Intent {
		return &Intent{Kind: "farm", Class: Filler, Urgency: 0.9, NeedsArea: Any, Reason: "level"}
	}}, FuncSource{N: "chore", F: func(View) *Intent {
		if w.now >= 20 {
			return &Intent{Kind: "repair", Class: Upkeep, Urgency: 0.1, NeedsArea: Any, Reason: "armor 30%"}
		}
		return nil
	}})
	for w.now = 0; w.now < 20; w.now += tick {
		if d := a.Decide(w.view()); d.Chosen.Source != "farm" {
			t.Fatalf("filler first: %v", d)
		}
	}
	if d := a.Decide(w.view()); d.Chosen.Source != "chore" {
		t.Fatalf("a filler must never starve upkeep: %v", d)
	}

	// an objective that keeps failing backs off exponentially, the filler runs with the reason
	w = newWorld()
	w.area = wood
	w.belt = 4
	a = New(DefaultConfig(), w.sources()...)
	var gaps []float64
	var failedAt float64
	for i := 0; i < 3; i++ {
		for ; ; w.now += tick {
			d := a.Decide(w.view())
			if d.Chosen.Source == "quest" {
				if i > 0 {
					gaps = append(gaps, w.now-failedAt)
				}
				a.Report(*d.Chosen, w.now, false)
				failedAt = w.now
				break
			}
			if i > 0 || w.now > 0 {
				found := false
				for _, r := range d.Rejected {
					found = found || (r.Intent.Source == "quest" && strings.Contains(r.Why, "backed off"))
				}
				if !found || d.Chosen.Class != Filler {
					t.Fatalf("want filler with the quest's back-off as reason: %v", d)
				}
			}
		}
	}
	t.Logf("quest retry gaps after failures 1 and 2: %v", gaps)
	if len(gaps) != 2 || gaps[0] != 30 || gaps[1] != 60 {
		t.Errorf("back-off should be 30s then 60s, got %v", gaps)
	}
}

func scenario() string {
	w := newWorld()
	a := New(DefaultConfig(), w.sources()...)
	var out []string
	for i := 0; i < 60; i++ {
		switch i {
		case 20:
			w.order = 40
		case 26:
			w.life = 0.2
		case 40:
			w.shopEmpty = false
		}
		d := a.Decide(w.view())
		out = append(out, d.String())
		w.exec(d.Chosen, a.Report)
	}
	return strings.Join(out, "\n") + "\n"
}

func TestDeterministicGolden(t *testing.T) {
	got := scenario()
	if got != scenario() {
		t.Fatal("two runs of the same scenario differ")
	}
	const path = "testdata/trace.golden"
	if *update {
		os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Errorf("trace differs from %s (run with -update after reviewing):\n%s", path, got)
	}
}

// Review 11, N3: the survive source needs a hostile near, with the safety fact as without it: a quiet hurt hero keeps
// the running trip. Both source sets.
func TestSurviveNeedsAHostile(t *testing.T) {
	for name, src := range map[string][]Source{"std": StdSources(), "upkeep": UpkeepSources()} {
		for _, f := range []map[string]int{
			{FactSafety: 1, FactLifePct: 40, FactTrip: 1},
			{FactSurviveBelow: 50, FactLifePct: 40, FactTrip: 1},
		} {
			if d := New(DefaultConfig(), src...).Decide(View{Area: "dark_wood", Facts: f}); d.Chosen == nil || d.Chosen.Source == SrcSurvive {
				t.Fatalf("%s %v: quiet hero, chosen %+v", name, f, d.Chosen)
			}
			f[FactHostile] = 2
			if d := New(DefaultConfig(), src...).Decide(View{Area: "dark_wood", Facts: f}); d.Chosen == nil || d.Chosen.Source != SrcSurvive {
				t.Fatalf("%s %v: hostile near, chosen %+v", name, f, d.Chosen)
			}
		}
	}
}
