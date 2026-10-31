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
