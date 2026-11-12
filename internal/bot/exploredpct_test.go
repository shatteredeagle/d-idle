package bot

import "testing"

func TestExploredThresholdClamp(t *testing.T) {
	for in, want := range map[int]int{0: 100, -5: 100, 101: 100, 100: 100, 60: 60, 30: 30, 10: 30} {
		if got := ExploredThreshold(in); got != want {
			t.Errorf("ExploredThreshold(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestExploredMath(t *testing.T) {
	if ExploredPercent(0, 0) != 0 || ExploredPercent(63, 100) != 63 || ExploredPercent(1, 3) != 33 || ExploredPercent(5, 4) != 100 {
		t.Fatal("ExploredPercent")
	}
	if ExploredReached(0, 0, 60) {
		t.Error("no points must never count as explored")
	}
	if !ExploredReached(60, 100, 60) || ExploredReached(59, 100, 60) {
		t.Error("threshold 60 edge")
	}
	if ExploredReached(99, 100, 0) || !ExploredReached(100, 100, 0) {
		t.Error("default threshold must need every point")
	}
}

type covMover struct {
	*fakeMover
	v, m int
}

func (c covMover) Coverage() (int, int) { return c.v, c.m }

func TestExploredEnoughByKnobAndRun(t *testing.T) {
	g, _, _ := repeatRig(t)
	g.r.d.Mover = covMover{fakeMover: g.mover, v: 70, m: 100}
	if g.r.exploredEnough(2) {
		t.Fatal("default 100: roam must run to the end")
	}
	g.r.e.Tune.Bot.ExploredPct = 60
	if !g.r.exploredEnough(2) {
		t.Fatal("60: 70 of 100 is enough")
	}
	g.r.e.Tune.Bot.ExploredPct = 80
	if g.r.exploredEnough(2) {
		t.Fatal("80: 70 of 100 is not enough")
	}
	g.r.e.Tune.Bot.ExploredPct = 60
	g.r.loop.run = &LevelRun{Area: 2, Kind: RunQuest}
	if g.r.exploredEnough(2) {
		t.Fatal("a quest run keeps roaming")
	}
}
