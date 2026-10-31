package sim_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/merc/sim"
)

var ctx = context.Background()

func kit() merc.Ctx {
	return merc.Ctx{Gold: 0, Kit: 200, Act: 2, PlayerClass: "Sorceress", HaveTP: true, HavePotions: true, Knobs: merc.DefaultKnobs(), FirstLutVisit: true}
}

// The planned life of a Sorceress's merc: free Rogue after Blood Raven, die, revive, strip and replace at Lut Gholein.
func TestFreeRogueThenDesertReplace(t *testing.T) {
	s := &sim.Sim{Gold: 2000, ClvlLevel: 12, FreeRogueLeft: true}
	st, _ := s.State()
	if d := merc.ShouldClaimFreeRogue(st, merc.FreeRogueCtx{AtKashya: true, BloodRavenDone: true}); !d.Do {
		t.Fatalf("claim: %+v", d)
	}
	if err := s.ClaimFreeRogue(); err != nil || s.Gold != 2000 {
		t.Fatalf("claim %v gold %d", err, s.Gold)
	}
	// dies in the field, the signal says leave
	if d := merc.MercDied(true, false, false); !d.LeaveArea {
		t.Fatal("no leave signal")
	}
	s.Kill()
	st, _ = s.State()
	c := kit()
	c.Gold = s.Gold
	d, price := merc.ShouldRevive(st, c, 0)
	if !d.Do {
		t.Fatalf("revive %+v price %d", d, price)
	}
	if paid, err := s.Revive(ctx); err != nil || paid != price || s.Gold != 2000-price {
		t.Fatalf("revive paid %d err %v gold %d", paid, err, s.Gold)
	}
	// gear on the Rogue, then Lut Gholein with money
	if err := s.Equip(ctx, merc.SlotHelm, merc.Item{Name: "Cap", Kind: merc.KindHelm, Identified: true, Defense: 20, W: 2, H: 2}); err != nil {
		t.Fatal(err)
	}
	s.Gold = 5000
	s.OpenWindow([]sim.Row{{merc.ClassDesert, "Might", 350, 9}, {merc.ClassDesert, "Defiance", 350, 9}})
	offers, _ := s.Offers()
	st, _ = s.State()
	c.Gold = s.Gold
	var pick *merc.HireOffer
	for i := range offers {
		if d := merc.ShouldReplace(st, offers[i], c); d.Do {
			if !d.StripFirst {
				t.Fatal("should strip first")
			}
			pick = &offers[i]
		}
	}
	if pick == nil || pick.Aura != "Defiance" {
		t.Fatalf("pick %+v", pick)
	}
	plan := merc.PlanStrip(st, 30)
	for _, step := range plan.Steps {
		if err := s.Unequip(ctx, step.Slot); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.Pack) != 1 || s.Pack[0].Name != "Cap" {
		t.Fatalf("pack %v", s.Pack)
	}
	before := s.Gold
	if paid, err := s.Hire(ctx, pick.Row); err != nil || paid != pick.Price || s.Gold != before-paid {
		t.Fatalf("hire %d %v", paid, err)
	}
	st, _ = s.State()
	if st.Class != merc.ClassDesert || st.Aura != "Defiance" {
		t.Fatalf("%+v", st)
	}
	// second visit: nothing better, no replace
	c.FirstLutVisit = false
	if d := merc.ShouldReplace(st, *pick, c); d.Do {
		t.Fatal("replaced again")
	}
}

// A paid hire first forfeits the free Rogue for good.
func TestPaidHireForfeitsFreeRogue(t *testing.T) {
	s := &sim.Sim{Gold: 3000, ClvlLevel: 10, FreeRogueLeft: true}
	s.OpenWindow([]sim.Row{{merc.ClassRogue, "Cold Arrow", 100, 3}})
	if _, err := s.Hire(ctx, 0); err != nil {
		t.Fatal(err)
	}
	s.Kill()
	if err := s.ClaimFreeRogue(); err == nil {
		t.Fatal("reward must be gone")
	}
}

func TestDeadMercStillForfeits(t *testing.T) {
	s := &sim.Sim{Gold: 3000, ClvlLevel: 10, FreeRogueLeft: true, Merc: merc.MercState{Owned: true, Class: merc.ClassDesert}}
	st, _ := s.State()
	if d := merc.ShouldClaimFreeRogue(st, merc.FreeRogueCtx{AtKashya: true, BloodRavenDone: true}); d.Do {
		t.Fatal("policy must not try")
	}
	if err := s.ClaimFreeRogue(); err == nil {
		t.Fatal("sim must refuse")
	}
}

func TestSimErrorsAndPotions(t *testing.T) {
	s := &sim.Sim{Gold: 10, ClvlLevel: 10}
	if _, err := s.Offers(); err != sim.ErrNoWindow {
		t.Error(err)
	}
	if _, err := s.Hire(ctx, 0); err != sim.ErrNoWindow {
		t.Error(err)
	}
	s.OpenWindow([]sim.Row{{merc.ClassDesert, "Defiance", 350, 9}})
	if _, err := s.Hire(ctx, 0); err != sim.ErrPoor {
		t.Error(err)
	}
	if _, err := s.Hire(ctx, 5); err != sim.ErrBadChoice {
		t.Error(err)
	}
	if _, err := s.Revive(ctx); err != sim.ErrNoMerc {
		t.Error(err)
	}
	s.Merc = merc.MercState{Owned: true, Alive: true, LifePct: 30}
	if err := s.GiveHealing(ctx); err != sim.ErrNoPotion {
		t.Error(err)
	}
	s.Potions = 1
	st, _ := s.State()
	if !merc.ShouldGiveHealing(st, merc.PotionCtx{HealingPotions: 1}, merc.DefaultKnobs()) {
		t.Error("should give")
	}
	if s.GiveHealing(ctx) != nil || s.Merc.LifePct != 80 {
		t.Errorf("life %d", s.Merc.LifePct)
	}
	s.Err = context.Canceled
	if _, err := s.State(); err == nil {
		t.Error("error passes through")
	}
	// a revive of a poor character fails and costs nothing
	s.Err = nil
	s.Kill()
	s.Merc.Level = 10
	s.Gold = 700
	if _, err := s.Revive(ctx); err != sim.ErrPoor || s.Gold != 700 {
		t.Error("poor revive")
	}
}
