package town

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/town/belt"
	"fmt"
	"strings"
)

// Stage 1 of the town planner (docs/design/fable-review-2026-11-21/town-planner.md, knob town.plan): the plan is solved
// beside every chores visit and logged, and nothing acts on it. PlanStage is the knob's value.
func (t *Town) PlanStage() int { return t.tune().Plan }

// legacyUntilStage is the town.plan stage from which the old gold knobs (bankFirst, withdrawShop) no longer decide:
// the purse stage replaces them. townflip-p5 section 6 lists the deletion that follows once that stage is the default.
const legacyUntilStage = 3

// townLegacy is true while the old gold counters are the path (town.plan below legacyUntilStage). Every read of
// bankFirst and withdrawShop goes through bankFirst and withdrawShop below, so deleting the knobs is a one-line change.
func (t *Town) townLegacy() bool { return t.PlanStage() < legacyUntilStage }

// bankFirst: deposit the surplus above the keep line before the shop (knob until the purse stage, then always).
func (t *Town) bankFirst() bool { return !t.townLegacy() || t.tune().BankFirst != 0 }

// withdrawShop: withdraw the shopping list when the pack cannot pay it (knob until the purse stage, then always).
func (t *Town) withdrawShop() bool { return !t.townLegacy() || t.tune().WithdrawShop != 0 }

// PlanShadow solves the supply plan of the current snapshot for the excursion profile of the chore plan (the `field`
// profile when the caller names none). It reads and changes nothing.
func (t *Town) PlanShadow(plan ChorePlan) (supply.Plan, error) {
	s, err := t.current()
	if err != nil {
		return supply.Plan{}, err
	}
	return t.planOf(s, plan)
}

func (t *Town) planOf(s *d2.Snapshot, plan ChorePlan) (supply.Plan, error) {
	pl, _, err := t.planCount(s, plan)
	return pl, err
}

// planCount is planOf plus the count it solved from (the viewer's budget row reads the purse and stash).
func (t *Town) planCount(s *d2.Snapshot, plan ChorePlan) (supply.Plan, supply.Count, error) {
	p := s.Player
	c := t.supplyCount(s)
	c.Purse, c.Stash, c.StashKnown = p.Gold, p.StashGold, p.StashGold > 0
	c.PortalCharges, c.HasTome = t.scrollCount(s, "tsc", "tbk")
	pl, err := solveShadow(p.Class, t.plan(p), belt.Slots(t.e.Data, s.Inventory), c, plan, &t.e.Tune.Loot, &t.e.Tune.Resource)
	return pl, c, err
}

// lastPlan is the plan of the last visit that solved one, kept for the viewer (GET /api/townplan).
type lastPlan struct {
	plan  supply.Plan
	count supply.Count
}

// PlanView is the Town plan card's data: the plan of the last visit that solved one (stage 1 and up of knob town.plan),
// the route stops of that visit; supply.NoView before any. Safe to call from the watch server.
func (t *Town) PlanView() supply.View {
	lp := t.lastPlan.Load()
	if lp == nil {
		return supply.NoView()
	}
	var stops []string
	if sp := t.lastStops.Load(); sp != nil {
		stops = *sp
	}
	return supply.ViewOf(lp.plan, lp.count, stops, 0)
}

// kitPackReserve is the healing potions a kit minimum counts in the pack beyond the belt: pack potions are not drinkable in a fight,
// so a floor above belt plus this only means a longer shop visit (V live 2026-11-24: a 4-slot belt, a floor of 9, three failed buys
// and the Andariel chunk parked on a kit it could not carry).
const kitPackReserve = 2

// solveShadow is planOf without the snapshot reads, so it runs on any class and band without the game tables.
func solveShadow(class d2.Class, band belt.Band, slots int, c supply.Count, plan ChorePlan, lt *env.LootTuning, r *env.ResourceTuning) (supply.Plan, error) {
	capHP, capMP := band.Capacity(belt.HP, slots), band.Capacity(belt.MP, slots)
	need := resource.ManaNeed(class)
	if capMP == 0 {
		need = 0 // the belt plan has no mana column: the kit asks for none
	}
	in := supply.Inputs{
		ManaNeed:   need,
		Capacity:   map[supply.Line]int{supply.Healing: capHP + min(lt.PotionBuffer, kitPackReserve), supply.Mana: capMP + lt.PotionBufferMana},
		BaseTarget: map[supply.Line]int{supply.Healing: capHP + lt.PotionBuffer, supply.Mana: capMP + lt.PotionBufferMana},
	}
	profile := plan.Profile
	if profile == "" {
		profile = "field"
	}
	n, err := supply.NeedFor("town", profile, in)
	if err != nil {
		return supply.Plan{}, err
	}
	prices := supply.Prices{supply.Healing: r.KitPotionGold, supply.Mana: r.KitPotionGold, supply.Portal: r.KitTPGold, supply.GoldCarry: 1}
	return supply.Solve(n, c, prices, supply.Policy{Reserve: plan.ReserveGold}, 0), nil
}

// logPlanShadow writes the stage 1 trace line when the knob is on (docs/design town-planner section 8).
func (t *Town) logPlanShadow(s *d2.Snapshot, plan ChorePlan) {
	if t.PlanStage() < 1 {
		return
	}
	pl, c, err := t.planCount(s, plan)
	if err != nil {
		t.e.Log.Warn("town plan (shadow): " + err.Error())
		return
	}
	t.lastPlan.Store(&lastPlan{plan: pl, count: c})
	t.lastStops.Store(nil) // the stops of the new visit arrive with its route (stage 5)
	t.e.Log.Info("town plan (shadow): " + pl.Trace())
}

// SupplyPlan solves the supply plan of a snapshot for an excursion profile; the bot's guards read it at stage 2 of the
// knob town.plan. It reads and changes nothing.
func (t *Town) SupplyPlan(s *d2.Snapshot, profile string, reserve int) (supply.Plan, error) {
	return t.planOf(s, ChorePlan{Profile: profile, ReserveGold: reserve})
}

// planLineTail is the end of the per-visit plan line (stage 3 and up): the lines the plan dropped with the gold each was
// short, the number of items the sell chore would offer and, for a short-gold plan, where the gold comes from next
// (scenarios gold-short-priorities and short-gold-handover read these fields).
func (t *Town) planLineTail(s *d2.Snapshot, pl supply.Plan) string {
	sellN := 0
	if lay, err := t.layout(s); err == nil {
		sellN = len(t.sellable(s, lay))
	}
	return planTail(pl, sellN, t.e.Tune.Loot.Vault != 0 && t.e.Tune.Loot.VaultSellPrompt != 0)
}

func planTail(pl supply.Plan, sellN int, vaultPrompt bool) string {
	var dropped []string
	for _, ln := range pl.Lines {
		if ln.Status == "dropped" || ln.Status == "short" {
			dropped = append(dropped, fmt.Sprintf("%s(%d short)", ln.Line, max(ln.Min-ln.Have-ln.Buy, 0)))
		}
	}
	out := fmt.Sprintf(" sell=%d", sellN)
	if len(dropped) > 0 {
		out += " dropped=" + strings.Join(dropped, ",")
	}
	if h := pl.Handover(supply.Gold{Sellable: sellN > 0, VaultPrompt: vaultPrompt}); len(h.Sources) > 0 {
		out += " hand-over: " + h.Reason
	}
	return out
}
