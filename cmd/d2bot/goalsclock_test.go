package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/scenario"
)

type virtClock struct{ t time.Time }

func (c *virtClock) Now() time.Time        { return c.t }
func (c *virtClock) Sleep(d time.Duration) { c.t = c.t.Add(d) }

// The d2d path hands the goal manager the env clock (card pgclock). Before the change Options.Clock was nil and the manager
// ran on time.Now: this test fails then (the manager's own clock would read the wall, not the virtual date).
func TestD2dGoalsOptionsUseTheEnvClock(t *testing.T) {
	virt := time.Date(1970, 1, 1, 0, 1, 30, 0, time.UTC) // lockstep's base is the epoch: nothing a wall clock could return
	c := &virtClock{t: virt}
	opt := d2dGoalsOptions(nil, "", nil, c, nil)
	if opt.Clock == nil {
		t.Fatal("no clock: goals.New would default to time.Now")
	}
	if got := opt.Clock(); !got.Equal(virt) {
		t.Fatalf("clock reads %v, want the env clock's %v", got, virt)
	}
	c.Sleep(45 * time.Second)
	if got := opt.Clock(); !got.Equal(virt.Add(45 * time.Second)) {
		t.Fatalf("clock did not follow the env clock: %v", got)
	}
}

// A manager built from those options records the virtual date (two builds' goals: caster boss farm, melee boss farm).
func TestManagerBuiltOnTheEnvClockStampsVirtualTime(t *testing.T) {
	for _, area := range []string{"Mephisto", "Andariel"} {
		c := &virtClock{t: time.Date(1970, 1, 1, 0, 0, 5, 0, time.UTC)}
		m, err := goals.New(d2dGoalsOptions(stubRunner{}, "", nil, c, nil))
		if err != nil {
			t.Fatal(err)
		}
		g, err := m.Add(goals.Goal{Type: goals.FarmArea, Params: goals.Params{Area: area, UntilMins: 1}}, -1)
		if err != nil {
			t.Fatal(err)
		}
		if !g.CreatedAt.Equal(c.t) {
			t.Errorf("%s: goal stamped %v, want the virtual %v", area, g.CreatedAt, c.t)
		}
	}
}

type stubRunner struct{}

func (stubRunner) Start(goals.Goal) error     { return nil }
func (stubRunner) Pause() error               { return nil }
func (stubRunner) Resume() error              { return nil }
func (stubRunner) Stop() error                { return nil }
func (stubRunner) Status() goals.Status       { return goals.Status{} }
func (stubRunner) Events() <-chan goals.Event { return nil }

// --- scripted taps inside the bot (card pgtaps step 2) ---

// lockClock is a lockstep stand-in: Sleep advances virtual time and then runs the hook, as LateClock does.
type lockClock struct {
	t    time.Time
	hook func()
	wall bool
}

func (c *lockClock) Now() time.Time { return c.t }
func (c *lockClock) OnSleep(fn func()) bool {
	if c.wall {
		return false
	}
	c.hook = fn
	return true
}
func (c *lockClock) Sleep(d time.Duration) {
	c.t = c.t.Add(d)
	if c.hook != nil {
		c.hook()
	}
}

type fakeDesk struct{ pressed []string }

func (d *fakeDesk) View() questboard.View {
	verb := "start"
	if len(d.pressed) > 0 {
		verb = "" // after the Start the board offers nothing
	}
	if verb == "" {
		return questboard.View{}
	}
	return questboard.View{Quests: []questboard.QuestRow{{ID: "q", Chunks: []questboard.ChunkRow{{ID: "c", Action: &questboard.ActionView{Verb: verb}}}}}}
}
func (d *fakeDesk) Start(q, c string) questboard.Reply {
	d.pressed = append(d.pressed, "start "+q+"/"+c)
	return questboard.Reply{OK: true}
}
func (d *fakeDesk) HandIn(q, c string) questboard.Reply {
	d.pressed = append(d.pressed, "handin "+q+"/"+c)
	return questboard.Reply{OK: true}
}

func scriptFile(t *testing.T) string {
	f := filepath.Join(t.TempDir(), "taps.json")
	if err := os.WriteFile(f, []byte(`{"quest":"q","every_s":30}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

// The tap lands on the stepping goroutine at the first grid point after the offer, whatever the wall speed: two runs that
// step the same ticks (one with a wall pause between steps) tap at equal virtual seconds. Must fail on the HTTP tapper's
// wall poll, which has no such property.
func TestScriptedTapsAreAFunctionOfSteppedTicks(t *testing.T) {
	var at []float64
	for i, wallPause := range []time.Duration{0, 3 * time.Millisecond} {
		clk, desk := &lockClock{t: time.Unix(0, 0)}, &fakeDesk{}
		f := scriptFile(t)
		if err := installScriptedTaps(clk, desk, f, slog.New(slog.DiscardHandler)); err != nil {
			t.Fatal(err)
		}
		for range 100 { // 100 sleeps of 1 virtual second
			clk.Sleep(time.Second)
			time.Sleep(wallPause)
		}
		if len(desk.pressed) != 1 || desk.pressed[0] != "start q/c" {
			t.Fatalf("run %d: pressed %v", i, desk.pressed)
		}
		b, err := os.ReadFile(f + ".taps.json")
		if err != nil {
			t.Fatal(err)
		}
		var recs []scenario.TapRec
		if err := json.Unmarshal(b, &recs); err != nil || len(recs) != 1 {
			t.Fatalf("taps.json %s: %v", b, err)
		}
		at = append(at, recs[0].At)
	}
	if at[0] != 30 || at[1] != at[0] {
		t.Fatalf("tap seconds %v, want 30 and equal", at)
	}
}

func TestScriptedTapsRefuseWhatTheyCannotDo(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	if err := installScriptedTaps(&lockClock{wall: true}, &fakeDesk{}, scriptFile(t), log); err == nil {
		t.Error("accepted on a wall clock")
	}
	if err := installScriptedTaps(&lockClock{}, nil, scriptFile(t), log); err == nil {
		t.Error("accepted without a desk")
	}
	if err := installScriptedTaps(&lockClock{}, &fakeDesk{}, filepath.Join(t.TempDir(), "none"), log); err == nil {
		t.Error("accepted a missing file")
	}
}
