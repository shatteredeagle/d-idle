package d2dsrc

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
)

// TestRealD2D builds one snapshot from a live d2d and prints a field-by-field report. Skipped unless
// D2D_SOCK names the control socket (private fork: ~/d2d-lab/fork/start.sh --start-screen ingame ...).
// It only reads. Run: D2D_SOCK=~/d2d-lab/fork/d2d.sock go test -count=1 -run TestRealD2D -v ./internal/d2dsrc
func TestRealD2D(t *testing.T) {
	sock := os.Getenv("D2D_SOCK")
	if sock == "" {
		t.Skip("D2D_SOCK not set")
	}
	c := d2dctl.New(sock, d2dctl.Options{Timeout: 8 * time.Second})
	defer c.Close()
	src := New(c, Options{Logf: t.Logf, Corpses: true, Warps: true})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := src.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("state=%v sources=%v", res.Snap.State, res.Report.Source)
	t.Logf("PRESENT (%d): %s", len(res.Report.Present), strings.Join(res.Report.Present, " "))
	var dk []string
	for k, v := range res.Report.Derived {
		dk = append(dk, k+" ("+v+")")
	}
	sort.Strings(dk)
	t.Logf("DERIVED (%d):\n  %s", len(dk), strings.Join(dk, "\n  "))
	for _, k := range res.Report.UnknownFields() {
		t.Logf("UNKNOWN %-28s %s", k, res.Report.Unknown[k])
	}
	if res.Snap.State == d2.StateInGame {
		if res.Snap.Player == nil {
			t.Fatal("in game but no player")
		}
		t.Logf("player %+v", *res.Snap.Player)
		t.Logf("monsters=%d objects=%d ground=%d items=%d", len(res.Snap.Monsters), len(res.Snap.Objects), len(res.Snap.Ground), res.Snap.Inventory.ItemCount)
	}
	if pr, err := src.Progress(ctx); err != nil {
		t.Logf("progress: %v", err)
	} else {
		t.Logf("progress: act1 quest words %v waypoints %#x", pr.Quest[0][:8], pr.Waypoints[0])
	}
}
