package loot_test

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/sim"
)

func skippedWhy(rep loot.PickReport, why string) bool {
	for _, s := range rep.Skipped {
		if s.Why == why {
			return true
		}
	}
	return false
}

func TestPoorCharacterTakesJunkGearToSell(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	dress(w)
	w.Player.Gold = 0
	sword := w.AddGround("ssd", 2, 6021, 4931) // plain short sword
	rep, err := l.PickUp(context.Background(), loot.PickOpts{})
	if err != nil || len(rep.Picked) != 1 || rep.Picked[0].ID != sword.ID {
		t.Fatalf("poor: %+v %v", rep, err)
	}
	if rep.Picked[0].Verdict.Action != loot.Sell {
		t.Fatalf("verdict %v", rep.Picked[0].Verdict)
	}
}

func TestHealthyCharacterLeavesJunkGear(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	dress(w)
	w.Player.Gold = r.Env.Tune.Loot.PoorGoldBelow + 1
	w.AddGround("ssd", 2, 6021, 4931)
	rep, _ := l.PickUp(context.Background(), loot.PickOpts{})
	if len(rep.Picked) != 0 || rep.Seen != 1 {
		t.Fatalf("healthy character picked junk: %+v", rep)
	}
	if l.EvaluateWith(w.AddGround("ssd", 2, 1, 1), loot.Ctx{CLvl: 8, Gold: 5000}).Action == loot.Sell {
		t.Fatal("junk sells when rich")
	}
}

func TestPoorEvaluateOnlySellsGear(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	poor := loot.Ctx{CLvl: 8, Gold: 0}
	if v := l.EvaluateWith(w.AddGround("ssd", 2, 1, 1), poor); v.Action != loot.Sell {
		t.Fatalf("junk sword while poor: %v", v)
	}
	if v := l.EvaluateWith(w.AddGround("tsc", 2, 1, 1), poor); v.Action == loot.Sell {
		t.Fatalf("scroll must not become junk to sell: %v", v)
	}
}

func goldPile(w interface {
	AddGround(string, int, int, int) d2.Unit
}, stats map[uint32]d2.Stats, amt int32) d2.Unit {
	u := w.AddGround("gld", 2, 6021, 4931)
	stats[u.Addr] = d2.Stats{d2.StatGold: amt}
	return u
}

func TestSmallGoldPileOnlySkippedWhenNotPoor(t *testing.T) {
	for _, tc := range []struct {
		gold     int
		wantSkip bool
	}{{0, false}, {1000, true}} {
		l, r := newLooter(t)
		w := r.W
		w.Player.Gold = tc.gold
		goldPile(w, w.Stats, 5)
		rep, _ := l.PickUp(context.Background(), loot.PickOpts{})
		if got := skippedWhy(rep, "small gold pile"); got != tc.wantSkip {
			t.Fatalf("gold %d: skipped=%v want %v (%+v)", tc.gold, got, tc.wantSkip, rep)
		}
	}
}

func TestPickReportSummary(t *testing.T) {
	rep := loot.PickReport{Seen: 3, Picked: []loot.Picked{{}}, GoldGained: 7,
		Skipped: []loot.Skipped{{Why: "no room"}, {Why: "no room"}}}
	got := rep.Summary()
	if !strings.Contains(got, "3 seen, picked 1, skipped 2 (2x no room), gold +7") {
		t.Fatal(got)
	}
}

func TestNakedCharacterKeepsUsableWeaponAndArmor(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	c := loot.Ctx{CLvl: 8, Gold: 5000, NeedWeapon: true, NeedBody: true, Str: 20, Dex: 20, Class: "sor"}
	if v := l.EvaluateWith(w.AddGround("lea", 2, 1, 1), c); v.Action != loot.Keep {
		t.Fatalf("leather armor for the empty body slot: %v", v)
	}
	if v := l.EvaluateWith(w.AddGround("ssd", 2, 1, 1), c); v.Action != loot.Keep {
		t.Fatalf("short sword for the empty weapon slot: %v", v)
	}
	c.NeedBody = false
	if v := l.EvaluateWith(w.AddGround("lea", 2, 1, 1), c); v.Action == loot.Keep {
		t.Fatalf("armor kept with the body slot filled: %v", v)
	}
	c.NeedBody, c.Str = true, 5
	if v := l.EvaluateWith(w.AddGround("lea", 2, 1, 1), c); v.Action == loot.Keep {
		t.Fatalf("armor kept without the strength: %v", v)
	}
	c.Str = 20
	if v := l.EvaluateWith(w.AddGround("hlm", 2, 1, 1), c); v.Action == loot.Keep {
		t.Fatalf("helm kept for the body slot: %v", v)
	}
}

// dress equips a weapon and a body armor, so the empty-slot rule stays out of tests about junk gear.
func dress(w *sim.World) {
	for _, e := range []struct {
		code string
		body int
	}{{"dgr", d2.BodyRHand}, {"qui", d2.BodyTorso}} {
		w.Add(e.code, d2.ContEquipment, 0, 0).Item.Body = e.body
	}
}

// Recovery (docs/recovery.md): the poor line rises to the kit gold, so a clvl 10 character at 400 gold still takes
// every small pile and sells junk gear; the totals feed the per-minute recovery log line.
func TestRecoveryTakesSmallPilesAndJunkUpToTheKitLine(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	dress(w)
	w.Player.Gold = 400 // above poorGoldBelow (300), under recoveryKitGold (500)
	goldPile(w, w.Stats, 5)
	w.AddGround("ssd", 2, 6030, 4931)
	rep, _ := l.PickUp(context.Background(), loot.PickOpts{})
	if len(rep.Picked) != 0 {
		t.Fatalf("not recovering: picked %+v", rep.Picked)
	}
	l.SetRecovering(true)
	rep, _ = l.PickUp(context.Background(), loot.PickOpts{})
	if len(rep.Picked) != 2 {
		t.Fatalf("recovering must take the pile and the sword: %+v", rep)
	}
	if p, g, _ := l.RecoveryTotals(); p != 2 || g <= 0 {
		t.Errorf("totals picked %d gold %d, want 2 and some gold", p, g)
	}
}

// A sell pickup leaves sellKeepExtra more cells free than other loot; while recovering it may use them (the sell trip
// is what frees the pack), otherwise a superior Cap stays on the ground at 90 percent full (G1 attempt 10).
func TestRecoverySellPickupUsesTheLastCells(t *testing.T) {
	for _, rec := range []bool{false, true} {
		l, r := newLooter(t)
		w := r.W
		dress(w)
		w.Player.Gold = 50
		for x := 0; x < 10; x++ {
			for y := 0; y < 4; y++ {
				if x >= 8 { // 8 free cells: 5 left after the sword, under FreeCellsKeep + sellKeepExtra (6)
					continue
				}
				w.Add("tsc", d2.ContInventory, x, y)
			}
		}
		w.AddGround("ssd", 2, 6021, 4931)
		l.SetRecovering(rec)
		rep, _ := l.PickUp(context.Background(), loot.PickOpts{})
		if got := len(rep.Picked) == 1; got != rec {
			t.Fatalf("recovering=%v: picked %d, skipped %+v", rec, len(rep.Picked), rep.Skipped)
		}
	}
}
