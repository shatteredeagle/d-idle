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
		if ok, err := l.RepairWanted(h, f.dur); err != nil {
			l.Fault(err)
		} else if ok && f.gold >= 10 {
			f.dur, f.gold = 100, f.gold-10
		}
	case "scrolls":
		if _, err := l.TPBuyWanted(h); err != nil {
			l.Fault(err)
		}
	}
	return ChoreInfo{Plan: plan}
}

func runT0(t *testing.T, f *fakeT0, c T0Config) []Result {
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
