package d2dloot_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/d2dloot"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/loot"
)

// TestLiveLoot starts a private d2d (autoloot off), makes monsters drop with "debug kill" on a few levels and lets the
// real looter pick up. Skipped unless D2D_BIN is set. The caller holds /tmp/d2d-lab.lock. Throwaway Sorceress, --no-save,
// d2d debug verbs only. Run: D2D_BIN=.../d2d.lab2 D2D_DATA=~/d2d-lab/data D2D_HOME=/tmp/x go test -count=1 -run LiveLoot -v ./internal/d2dloot
func TestLiveLoot(t *testing.T) {
	bin, dd := os.Getenv("D2D_BIN"), os.Getenv("D2D_DATA")
	if bin == "" || dd == "" {
		t.Skip("D2D_BIN / D2D_DATA not set")
	}
	home := os.Getenv("D2D_HOME")
	db, err := data.Open(dd, filepath.Join(home, "tables"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
	defer cancel()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tune := env.DefaultTuning()
	be, err := d2denv.Build(ctx, d2denv.Config{ID: "dl", Log: log, Tune: &tune, Data: db, Sock: filepath.Join(home, "d.sock"), LinkWait: 30 * time.Second,
		Bin: bin, DataDir: dd, Home: home, Nice: 10, LogFile: filepath.Join(home, "d2d.out"),
		ExtraArgs: []string{"--toggle", "autoloot=off,autoparty=off", "--lockstep"},
		Spec:      d2dinput.Spec{Name: "Lootie", Class: 4, Seed: 7, NoSave: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	if err := be.Start(ctx, 0); err != nil {
		t.Fatal(err)
	}
	e := be.Env
	lt := loot.New(e, be.Navigator())
	d2dloot.Attach(lt, be.Source, db)
	dbg := func(a ...string) {
		if _, err := be.Client.Debug(ctx, a...); err != nil {
			t.Logf("debug %v: %v", a, err)
		}
	}
	dbg("stat", "12", "8")
	var picked, gold, skipped int
	for _, lv := range []string{"2", "3", "4", "2"} {
		dbg("goto", lv)
		e.Clock.Sleep(4 * time.Second)
		dbg("kill")
		e.Clock.Sleep(2 * time.Second)
		s, _ := e.Snapshot()
		t.Logf("level %s: ground %d, gold %d, pack %d items", lv, len(s.Ground), s.Player.Gold, s.Inventory.ItemCount)
		for round := 0; round < 3; round++ {
			rep, err := lt.PickUp(ctx, loot.PickOpts{Radius: 80})
			t.Logf("round %d: %s err=%v full=%v", round, rep.Summary(), err, rep.Full)
			for _, p := range rep.Picked {
				t.Logf("  picked %s (%s) gold+%d", p.Name, p.Verdict, p.Gold)
			}
			for _, sk := range rep.Skipped {
				t.Logf("  skipped %s: %s", sk.Name, sk.Why)
			}
			for _, sk := range rep.Failed {
				t.Logf("  FAILED %s: %s", sk.Name, sk.Why)
			}
			picked += len(rep.Picked)
			gold += rep.GoldGained
			skipped += len(rep.Skipped)
			if rep.Seen == 0 || err != nil {
				break
			}
		}
	}
	s, _ := e.Snapshot()
	t.Logf("TOTAL picked %d, gold gained %d, skipped %d; purse %d, pack %d items, left behind %d", picked, gold, skipped, s.Player.Gold, s.Inventory.ItemCount, lt.LeftBehind())
}
