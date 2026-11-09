package main

import (
	"path/filepath"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/watch"
)

// End to end for card previewfeed: fake fights through the fighter's hook become close events, a restart keeps them,
// and the stance preview of the policy sheet turns them into real deltas for two classes. Fixtures only.
func TestFightFeedToPolicyPreview(t *testing.T) {
	lows := []int{30, 40, 60, 58, 70, 28, 90, 24, 52, 21}
	for _, class := range []string{"paladin", "sorceress"} {
		dir := t.TempDir()
		now := 1000.0
		mkAway := func() *watch.AwayLog {
			a := watch.NewAwayLog(func() float64 { return now })
			if err := a.AttachSpool(filepath.Join(dir, "away-spool.jsonl")); err != nil {
				t.Fatal(err)
			}
			return a
		}
		rig := &ladderRig{Fun: &watch.Fun{Away: mkAway()}}
		f := &combat.Fighter{}
		useFightFeed(rig, f, "", nil) // no state dir: spool stays as attached above
		if f.OnFight == nil {
			t.Fatal("hook not installed")
		}
		for i, l := range lows {
			now = float64(100 + 50*i)
			f.OnFight(combat.FightEnd{Area: 2, MinLifePct: l, Sec: 9, Potions: i % 2, Damage: 40})
		}
		now = 1000
		// restart: a fresh log reads the spool
		away := mkAway()
		if n := len(away.Events()); n != len(lows) {
			t.Fatalf("%s: %d events after restart, want %d", class, n, len(lows))
		}
		cat, _ := decide.Shipped()
		tbl, err := decide.ShippedPolicy(cat)
		if err != nil {
			t.Fatal(err)
		}
		st, err := decide.Open(t.TempDir(), "pf", cat, &decide.ManualClock{})
		if err != nil {
			t.Fatal(err)
		}
		ps, err := watch.NewPolicySheet(st, cat, tbl, func() string { return class }, func() float64 { return now }, away, func(map[string]*float64) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		ps.SetStarts(func(string) (map[string]string, error) { return nil, nil })
		for _, opt := range []string{"cautious", "aggressive"} {
			res, err := ps.Preview("stance", opt, 0)
			if err != nil {
				t.Fatalf("%s/%s: %v", class, opt, err)
			}
			if !res.Supported || !res.Enough || res.Window.Events != len(lows) {
				t.Fatalf("%s/%s: %+v", class, opt, res)
			}
			moved := false
			for _, l := range res.Lines {
				moved = moved || (l.Delta != nil && *l.Delta != 0)
			}
			if !moved {
				t.Errorf("%s/%s: no delta from ten recorded fights: %+v", class, opt, res.Lines)
			}
		}
	}
}
