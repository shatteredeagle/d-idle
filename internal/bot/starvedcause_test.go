package bot

import (
	"strings"
	"testing"

	"d2lod-bot/internal/arbiter"
)

// arbiter-starved used to say only "level/sweep: nothing else wanted: level the hero". It now names the winner, the area
// and the parked board runs, for two different starved sources.
func TestStarvedTextNamesTheCause(t *testing.T) {
	win := &arbiter.Intent{Source: arbiter.SrcQuest, Kind: "step", Reason: "quest step in Dark Wood"}
	for _, c := range []struct {
		in   arbiter.Intent
		want []string
	}{
		{arbiter.Intent{Source: arbiter.SrcLevel, Kind: "sweep", Reason: "nothing else wanted: level the hero"}, []string{"outranked by " + win.Key(), "quest step in Dark Wood", "in Dark Wood", "quest board: search_for_cain/scroll paused_after_deaths"}},
		{arbiter.Intent{Source: arbiter.SrcRecovery, Kind: "farm_gold", Reason: "broke: recovery farm"}, []string{"recovery/farm_gold", "outranked by " + win.Key()}},
	} {
		got := starvedText(c.in, arbiter.Decision{Chosen: win, Area: "Dark Wood"}, "search_for_cain/scroll paused_after_deaths")
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("missing %q in %q", w, got)
			}
		}
	}
	if got := starvedText(arbiter.Intent{Source: arbiter.SrcLevel, Kind: "sweep", Reason: "r"}, arbiter.Decision{}, ""); !strings.Contains(got, "chose nothing") {
		t.Errorf("no winner: %q", got)
	}
}

// The standing leveling filler is never reported as starved; a temporary filler and any other class still are.
func TestStandingFillerIsNotStarved(t *testing.T) {
	sweep := arbiter.Intent{Source: arbiter.SrcLevel, Kind: "sweep", Class: arbiter.Filler}
	weak := arbiter.Intent{Source: arbiter.SrcLevel, Kind: "level_up", Class: arbiter.Filler, TTL: 900}
	farm := arbiter.Intent{Source: arbiter.SrcRecovery, Kind: "farm_gold", Class: arbiter.Objective}
	if !standingFiller(sweep) || standingFiller(weak) || standingFiller(farm) {
		t.Fatal("only a filler without a TTL is standing")
	}
	r := &Runner{}
	r.noteStarvedAll(arbiter.Decision{Starved: []arbiter.Intent{sweep, weak, farm}})
	if got := len(r.inv.pending); got != 2 {
		t.Fatalf("%d starved hits, want 2 (the sweep is dropped): %+v", got, r.inv.pending)
	}
	for _, h := range r.inv.pending {
		if strings.Contains(h.Detail, "level/sweep") {
			t.Errorf("the standing filler was reported: %s", h.Detail)
		}
	}
}

// L6 (live review 2026-11-28): a leveling filler outranked by the quest step is not starved while that step is the
// board's training slice; outranked by a real quest step, or another source outranked, it still is.
func TestTrainingStepDoesNotStarveTheLevelingFiller(t *testing.T) {
	g, _ := boardRig(t, true)
	// A temporary filler: the standing sweep is never reported at all (TestStandingFillerIsNotStarved).
	level := arbiter.Intent{Source: arbiter.SrcLevel, Kind: "level_up", Class: arbiter.Filler, TTL: 900, Reason: "refused: too weak, level first"}
	d := arbiter.Decision{Chosen: &arbiter.Intent{Source: arbiter.SrcQuest, Kind: "step", Reason: "quest step in Tamoe Highland"}, Starved: []arbiter.Intent{level}}
	hits := func() int {
		g.r.inv.mu.Lock()
		defer g.r.inv.mu.Unlock()
		n := len(g.r.inv.pending)
		g.r.inv.pending = nil
		return n
	}
	g.r.noteStarvedAll(d)
	if n := hits(); n != 1 {
		t.Fatalf("a real quest step outranks the filler for the limit: want one hit, got %d", n)
	}
	g.r.boardTraining.Store(true) // set by boardTrain, cleared when the board runs or awaits a chunk
	g.r.noteStarvedAll(d)
	if n := hits(); n != 0 {
		t.Fatalf("the quest step is the training slice: want no hit, got %d", n)
	}
	d.Starved = []arbiter.Intent{{Source: arbiter.SrcGoldFarm, Kind: "farm", Class: arbiter.Objective, Reason: "gold"}}
	g.r.noteStarvedAll(d)
	if n := hits(); n != 1 {
		t.Fatalf("another source is still starved while training: got %d", n)
	}
}
