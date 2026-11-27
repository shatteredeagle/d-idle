package town

import (
	"sort"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/town/belt"
)

// Stage 4 of the town planner, sell side (docs/design/fable-review-2026-11-21/town-planner.md 5.2, knob town.plan >= 4):
// the sell chore never sells a pack potion the supply plan needs. Per line the hero keeps max(target, min) potions,
// belt plus pack; the belt is never sold, so the pack keeps the rest, best tier first, and a lower tier (a potion under
// the tier floor among them) is the surplus that goes first. Rejuvenations stay with the rejuvenation cap.

// packPotion is one healing or mana potion in the pack.
type packPotion struct {
	ID   uint32
	Kind belt.Kind
	Tier int
}

// planKeepOf is how many pack potions of each kind the plan keeps: the line's max(target, min) minus what the belt (and,
// for healing, every rejuvenation) already covers.
func planKeepOf(pl supply.Plan, c supply.Count) map[belt.Kind]int {
	keep := map[belt.Kind]int{}
	for _, ln := range pl.Lines {
		want := max(ln.Target, ln.Min)
		switch ln.Line {
		case supply.Healing:
			keep[belt.HP] = max(0, want-c.Belt[supply.HP]-c.Belt[supply.RV]-c.Pack[supply.RV])
		case supply.Mana:
			keep[belt.MP] = max(0, want-c.Belt[supply.MP])
		}
	}
	return keep
}

// keepByTier picks the pack potions a keep count protects, the best tier of each kind first (ties by id, so the
// choice is the same every visit).
func keepByTier(ps []packPotion, keep map[belt.Kind]int) map[uint32]bool {
	sorted := append([]packPotion(nil), ps...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Tier != sorted[j].Tier {
			return sorted[i].Tier > sorted[j].Tier
		}
		return sorted[i].ID < sorted[j].ID
	})
	out := map[uint32]bool{}
	n := map[belt.Kind]int{}
	for _, p := range sorted {
		if n[p.Kind] < keep[p.Kind] {
			n[p.Kind]++
			out[p.ID] = true
		}
	}
	return out
}

// planSellKeep is the set of pack potions the sell chore must not sell at town.plan >= 4, nil below stage 4 or when the
// plan cannot be solved (then today's rules decide, and the failure is logged).
func (t *Town) planSellKeep(s *d2.Snapshot) map[uint32]bool {
	if t.PlanStage() < 4 || s == nil || s.Player == nil || s.Inventory == nil {
		return nil
	}
	pl, err := t.planOf(s, ChorePlan{Profile: t.visitProfile})
	if err != nil {
		t.e.Log.Warn("town plan (sell): " + err.Error())
		return nil
	}
	var ps []packPotion
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		b := t.base(u)
		if b == nil {
			continue
		}
		if k := belt.PotionKind(b); k == belt.HP || k == belt.MP {
			ps = append(ps, packPotion{ID: u.ID, Kind: k, Tier: belt.Tier(b)})
		}
	}
	return keepByTier(ps, planKeepOf(pl, t.supplyCount(s)))
}

// pickupKeep is the stage 4 pickup side (town-planner 5.3, installed on the shared Looter by New and SetJudge): the pack
// keep per kind of the plan of the excursion the last visit supplied, so the pickup fills the pack to the number the
// sell chore protects. ok false below stage 4 or when the plan cannot be solved (the looter keeps its buffers).
func (t *Town) pickupKeep(s *d2.Snapshot) (map[belt.Kind]int, bool) {
	if t.PlanStage() < 4 || s == nil || s.Player == nil || s.Inventory == nil {
		return nil, false
	}
	profile := ""
	if p := t.excursion.Load(); p != nil {
		profile = *p
	}
	pl, err := t.planOf(s, ChorePlan{Profile: profile})
	if err != nil {
		msg := err.Error()
		if old := t.pickupErr.Swap(&msg); old == nil || *old != msg { // once per distinct error: the looter asks per ground potion
			t.e.Log.Warn("town plan (pickup): " + msg)
		}
		return nil, false
	}
	return planKeepOf(pl, t.supplyCount(s)), true
}

// shareLooter installs the town's stage 4 pickup rule on a Looter the town judges with.
func (t *Town) shareLooter(j Judge) {
	if l, ok := j.(*loot.Looter); ok {
		l.SetPotionPlan(t.pickupKeep)
	}
}
