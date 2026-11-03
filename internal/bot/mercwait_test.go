package bot

import (
	"context"
	"testing"
)

func TestWaitForMercHook(t *testing.T) {
	old := MercWait
	defer func() { MercWait = old }()
	r := &Runner{}
	MercWait = nil
	r.waitForMerc(context.Background(), "portal") // nil hook: nothing happens
	var got []string
	MercWait = func(_ context.Context, why string) { got = append(got, why) }
	r.waitForMerc(context.Background(), "waypoint")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.waitForMerc(ctx, "portal") // cancelled: no wait
	if len(got) != 1 || got[0] != "waypoint" {
		t.Errorf("hook calls %v", got)
	}
}
