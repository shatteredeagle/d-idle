package drill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/ladder"
)

// fakeT0 is a T0Lab whose chores ask the installed ladder the way the town call sites do.
type fakeT0 struct {
	TownLab
	tick       int
	gold, dur  int
	starts     []string
	cheatAfter bool // a cheat sneaks in after the start (the drill must call it CHEATED)
	writes     int
	class      d2.Class
}

func (f *fakeT0) VirtualNow() time.Duration { return time.Duration(f.tick) * 40 * time.Millisecond }
func (f *fakeT0) Tick(_ context.Context, n int) error {
	f.tick += n
	return nil
}
func (f *fakeT0) ClearPack(_ context.Context, r *Run) error {
	_, err := r.Cheat(context.Background(), "debug clearinv", "empty pack")
	return err
}
func (f *fakeT0) SetGold(_ context.Context, r *Run, n int) error {
	f.gold = n
	_, err := r.Cheat(context.Background(), "debug stat 14", "gold")
	return err
}
func (f *fakeT0) Wear(_ context.Context, r *Run) error {
	f.dur = 1
	_, err := r.Cheat(context.Background(), "debug durability all 1", "wear")
	return err
}
func (f *fakeT0) TView(context.Context) (TownView, error) {
	return TownView{InTown: true, Gold: f.gold, DurCur: f.dur, DurMax: 100}, nil
}
func (f *fakeT0) StartRun(_ context.Context, run, _ string) error {
	f.starts = append(f.starts, run)
	return nil
}
func (f *fakeT0) WritesAfterStart(string) (int, error) { return f.writes, nil }
func (f *fakeT0) Chores(_ context.Context, plan string) ChoreInfo {
	h := ladder.Hero{Class: f.class, AreaClass: "town", Gold: f.gold, Durability: -1}
	l := ladder.Active()
	if l == nil {
		return ChoreInfo{Plan: plan, Err: "no ladder installed"}
	}
	switch plan {
	case "repair":
		if ok, err := l.RepairWanted(h, f.dur, f.dur < 30); err != nil {
			l.Fault(err)
		} else if ok && f.gold >= 10 {
			f.dur, f.gold = 100, f.gold-10
		}
	case "scrolls":
		if _, err := l.TPBuyWanted(h, f.gold >= 100); err != nil {
			l.Fault(err)
		}
		if _, err := l.RestockWanted(h, ladder.BeltLow, f.gold, f.gold >= 300); err != nil {
			l.Fault(err)
		}
	}
	return ChoreInfo{Plan: plan}
}

// fakeFight is a fakeT0 that also drives the fight-time points: each pass asks the ladder with a state that moves with
// the pass count, the way the real call sites ask at changing life, gold and level.
type fakeFight struct {
	*fakeT0
	passes map[string]int
}

func (f *fakeFight) FightPoints(_ context.Context, point string) ChoreInfo {
	l := ladder.Active()
	if l == nil {
		return ChoreInfo{Plan: point, Err: "no ladder installed"}
	}
	n := f.passes[point]
	f.passes[point]++
	h := ladder.Hero{Class: f.class, AreaClass: "field", Gold: f.gold, Durability: -1}
	life := 10 + 9*(n%10)
	kit := []string{ladder.KitOK, ladder.KitThin, ladder.KitNone}[n%3]
	var err error
	switch point {
	case ladder.PointDrink:
		_, err = l.DrinkChoice(h, life, ladder.HeldFor(n%2, 0), kit, n%4 == 0, "fight")
	case ladder.PointUnique:
		_, err = l.UniqueFight(h, life, 1+n%3, kit, "fight")
	case ladder.PointTooHard:
		_, err = l.TooHard(h, 12+n%5, []string{"alive", "dead", "none"}[n%3], kit, "skip")
	case ladder.PointMerc:
		_, err = l.MercHire(h, []string{"dead", "none"}[n%2], kit, f.gold >= 400)
	}
	if err != nil {
		l.Fault(err)
	}
	return ChoreInfo{Plan: point}
}

// The fight-time points run in the drill every round when the lab offers them, for 5 profiles x 2 builds, and the
// record counts the passes.
func TestT0DrillRunsTheFightPoints(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		c := DefaultT0()
		c.Minutes = 10
		f := &fakeFight{fakeT0: &fakeT0{class: cl}, passes: map[string]int{}}
		res := runT0(t, f, c)
		for _, x := range res {
			if x.Verdict != Pass {
				t.Errorf("%v %s: %s %s | %s", cl, x.ID, x.Verdict, x.Reason, x.Observed)
			}
			if !strings.Contains(x.Observed, `"fightPasses":80`) {
				t.Errorf("%s: record lacks 80 fight passes (20 rounds x 4 points): %s", x.ID, x.Observed)
			}
		}
		for _, pt := range T0FightPoints {
			if f.passes[pt] < 20 {
				t.Errorf("%v: %s asked %d times", cl, pt, f.passes[pt])
			}
		}
	}
}

func runT0(t *testing.T, f T0Lab, c T0Config) []Result {
	t.Helper()
	r := &Runner{W: &fakeWorld{}, Repeat: 1}
	g := r.RunGroup(context.Background(), T0(f, c))
	return g.Results
}

func TestT0DrillAllProfilesTwoBuilds(t *testing.T) {
	for _, cl := range []d2.Class{d2.Sorceress, d2.Paladin} {
		c := DefaultT0()
		c.Minutes = 10
		res := runT0(t, &fakeT0{class: cl}, c)
		if len(res) != 5 {
			t.Fatalf("%d drills, want 3 known profiles and 2 holdouts", len(res))
		}
		for _, x := range res {
			if x.Verdict != Pass {
				t.Errorf("%v %s: %s %s | %s", cl, x.ID, x.Verdict, x.Reason, x.Observed)
			}
			if !strings.Contains(x.Observed, "writes after start: 0") {
				t.Errorf("%s: record lacks the writes line: %s", x.ID, x.Observed)
			}
		}
	}
}

func TestT0DrillCallsCheatedRuns(t *testing.T) {
	res := runT0(t, &fakeT0{class: d2.Sorceress, writes: 2}, T0Config{Profiles: []string{"cautious"}, Minutes: 1})
	if res[0].Verdict != Fail || !strings.Contains(res[0].Reason, "CHEATED") {
		t.Fatalf("%+v", res[0])
	}
}

func TestT0DrillFileProfileAndBadInputs(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	if err := os.WriteFile(p, []byte(`{"rows":[{"point":"t1.repair","answer":"skip","delaySec":2}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &fakeT0{class: d2.Sorceress}
	res := runT0(t, f, T0Config{Profiles: []string{"file"}, Answers: p, Minutes: 5})
	if res[0].Verdict != Pass || len(f.starts) == 0 {
		t.Fatalf("%+v starts %v", res[0], f.starts)
	}
	if res := runT0(t, &fakeT0{}, T0Config{Profiles: []string{"file"}, Answers: filepath.Join(dir, "none.json")}); res[0].Verdict != Blocked {
		t.Fatalf("missing answer file: %+v", res[0])
	}
	if res := runT0(t, &fakeT0{}, T0Config{Profiles: []string{"reckless"}}); res[0].Verdict != Blocked {
		t.Fatalf("unknown profile: %+v", res[0])
	}
}
