package town

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/town/belt"
)

// Stage 4 sell keep (town-planner 5.2) on two builds: the pack keeps what the line's target still lacks after the belt,
// the best tier first, and the rest is surplus. No game tables.
func TestPlanSellKeepTwoBuilds(t *testing.T) {
	lt, rs := env.DefaultLootTuning(), env.DefaultResourceTuning()
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		band := belt.PlanForClass(class, 20)
		capHP, capMP := band.Capacity(belt.HP, belt.Columns), band.Capacity(belt.MP, belt.Columns)
		var c supply.Count
		c.Belt[supply.HP], c.Belt[supply.MP] = capHP, capMP
		c.Purse, c.StashKnown, c.PortalCharges, c.HasTome = 2000, true, 5, true
		pl, err := solveShadow(class, band, belt.Columns, c, ChorePlan{Profile: "monastery_road"}, &lt, &rs)
		if err != nil {
			t.Fatalf("%v: %v", class, err)
		}
		keep := planKeepOf(pl, c)
		// Belt full: the healing target is clamped to belt plus the pack reserve, so the pack keeps the reserve.
		if got, want := keep[belt.HP], min(lt.PotionBuffer, kitPackReserve); got != want {
			t.Errorf("%v: pack healing keep %d, want %d (belt full): %s", class, got, want, pl.Trace())
		}
		manaLine := pl.Need.Lines[supply.Mana].Target > 0
		if manaLine != (keep[belt.MP] > 0) || manaLine && keep[belt.MP] != lt.PotionBufferMana {
			t.Errorf("%v: pack mana keep %d (mana line %v, capMP %d): %s", class, keep[belt.MP], manaLine, capMP, pl.Trace())
		}

		// Pack: healing potions of tiers 1 to 4 plus more than the keep; the best tiers stay, tier 1 goes first.
		var ps []packPotion
		for i := range keep[belt.HP] + 2 {
			ps = append(ps, packPotion{ID: uint32(10 + i), Kind: belt.HP, Tier: 1 + i%4})
		}
		kept := keepByTier(ps, keep)
		if len(kept) != keep[belt.HP] {
			t.Fatalf("%v: kept %d, want %d", class, len(kept), keep[belt.HP])
		}
		for _, p := range ps {
			if p.Tier == 1 && kept[p.ID] && keep[belt.HP] < len(ps)-1 {
				t.Errorf("%v: a tier 1 potion was kept before better ones: %v", class, kept)
			}
		}
	}
}

// An empty belt raises the pack keep to the whole target, and a rejuvenation counts toward the healing line.
func TestPlanKeepEmptyBeltAndRejuv(t *testing.T) {
	pl := supply.Plan{Lines: []supply.PlanLine{{Line: supply.Healing, Min: 4, Target: 6}, {Line: supply.Mana, Min: 2, Target: 1}}}
	var c supply.Count
	c.Pack[supply.RV] = 1
	keep := planKeepOf(pl, c)
	if keep[belt.HP] != 5 || keep[belt.MP] != 2 {
		t.Fatalf("keep %v, want HP 5 (target 6 minus one rejuvenation), MP 2 (min above target)", keep)
	}
	if got := keepByTier([]packPotion{{1, belt.MP, 3}, {2, belt.MP, 3}, {3, belt.MP, 3}}, keep); !got[1] || !got[2] || got[3] {
		t.Fatalf("equal tiers keep the lowest ids: %v", got)
	}
}
