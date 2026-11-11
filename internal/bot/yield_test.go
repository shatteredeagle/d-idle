package bot

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

func yieldFixture(town bool, clearErr error) (YieldDeps, *int) {
	area := 3
	if town {
		area = 1
	}
	calls := 0
	snap := &d2.Snapshot{Player: &d2.Player{Area: area}}
	return YieldDeps{
		Snapshot: func() (*d2.Snapshot, error) { return snap, nil },
		Friendly: func(_ *d2.Snapshot, u d2.Unit) bool { return u.ID == 99 },
		Clear:    func(context.Context, int, time.Duration) error { calls++; return clearErr },
	}, &calls
}

func TestPackYield(t *testing.T) {
	hostile := []d2.Unit{{ID: 99}, {ID: 5}}
	boom := errors.New("left")
	cases := []struct {
		name  string
		town  bool
		near  []d2.Unit
		cerr  error
		calls int
		want  error
	}{
		{"hostile clears", false, hostile, nil, 1, nil},
		{"town skips", true, hostile, nil, 0, nil},
		{"friendly only skips", false, []d2.Unit{{ID: 99}}, nil, 0, nil},
		{"clear timeout replans", false, hostile, context.DeadlineExceeded, 1, nil},
		{"fatal propagates", false, hostile, boom, 1, boom},
	}
	for _, c := range cases {
		d, calls := yieldFixture(c.town, c.cerr)
		if err := PackYield(d)(context.Background(), c.near); !errors.Is(err, c.want) || *calls != c.calls {
			t.Errorf("%s: err=%v calls=%d, want %v %d", c.name, err, *calls, c.want, c.calls)
		}
	}
	d, _ := yieldFixture(false, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := PackYield(d)(ctx, hostile); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx: %v", err)
	}
}

// A pack of five where each round kills up to three: the hand-over keeps fighting until the pack is dead instead of
// returning after the first short round (class free: the pack and the kills are all the rule reads, run for a
// Paladin and a Sorceress snapshot).
func TestPackYieldKeepsFightingUntilThePackIsDead(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		alive := 5
		units := func() []d2.Unit {
			var us []d2.Unit
			for i := 0; i < alive; i++ {
				us = append(us, d2.Unit{ID: uint32(10 + i), Type: d2.TypeMonster, X: 105, Y: 100, Mode: d2.MonModeStand})
			}
			return us
		}
		rounds := 0
		d := YieldDeps{
			Snapshot: func() (*d2.Snapshot, error) {
				return &d2.Snapshot{Player: &d2.Player{Area: 3, Class: class, X: 100, Y: 100}, Monsters: units()}, nil
			},
			Friendly: func(*d2.Snapshot, d2.Unit) bool { return false },
			Clear: func(context.Context, int, time.Duration) error {
				rounds++
				alive = max(0, alive-3)
				return context.DeadlineExceeded // each round hits its short timeout
			},
		}
		if err := PackYield(d)(context.Background(), units()[:1]); err != nil {
			t.Fatal(err)
		}
		if alive != 0 || rounds != 2 {
			t.Errorf("%v: alive %d after %d rounds, want 0 after 2", class, alive, rounds)
		}
	}
}

// Monsters that never die (unreachable, ignored by the fighter) end the hand-over after two rounds without a kill.
func TestPackYieldStopsWhenNothingDies(t *testing.T) {
	rounds := 0
	d := YieldDeps{
		Snapshot: func() (*d2.Snapshot, error) {
			return &d2.Snapshot{Player: &d2.Player{Area: 3, X: 100, Y: 100}, Monsters: []d2.Unit{{ID: 5, Type: d2.TypeMonster, X: 105, Y: 100, Mode: d2.MonModeStand}}}, nil
		},
		Friendly: func(*d2.Snapshot, d2.Unit) bool { return false },
		Clear:    func(context.Context, int, time.Duration) error { rounds++; return nil },
	}
	if err := PackYield(d)(context.Background(), []d2.Unit{{ID: 5}}); err != nil {
		t.Fatal(err)
	}
	if rounds != 2 { // two rounds without a kill stop it
		t.Errorf("rounds %d, want 2", rounds)
	}
}
