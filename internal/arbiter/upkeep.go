package arbiter

import (
	"fmt"

	"d2lod-bot/internal/supply"
)

// town10 (docs/design/fable-review-2026-11-21/town-planner.md section 6): one upkeep source reads the supply plan and
// replaces the trip and resupply sources. It is not part of StdSources: the shadow keeps today's two sources until the
// arbiter controls the chores family (bot.arbiterControl, bot.arbiterFamilies bit 4); UpkeepSources is the set a
// controlling arbiter runs. No class or build literal: the build's needs arrive through the plan facts.

const SrcUpkeep = "upkeep"

// Facts the upkeep source reads, filled by PlanFacts.
const (
	FactPlanBuy     = "plan_buy"      // units the plan would buy in this visit
	FactPlanVerdict = "plan_verdict"  // PlanReady, PlanLean, PlanShort or PlanBlocked
	FactPlanShort   = "plan_short"    // gold missing for the first open min line
	FactPlanMinOpen = "plan_min_open" // 1 while a line holds less than its min
)

const (
	PlanReady = iota
	PlanLean
	PlanShort
	PlanBlocked
)

// PlanFacts turns a plan into the facts of the upkeep source (the caller merges them into the View).
func PlanFacts(p supply.Plan) map[string]int {
	f := map[string]int{FactPlanShort: p.Short}
	switch p.Verdict {
	case supply.ReadyLean:
		f[FactPlanVerdict] = PlanLean
	case supply.ShortGold:
		f[FactPlanVerdict] = PlanShort
	case supply.Blocked:
		f[FactPlanVerdict] = PlanBlocked
	}
	for _, ln := range p.Lines {
		f[FactPlanBuy] += ln.Buy
		if ln.Have < ln.Min {
			f[FactPlanMinOpen] = 1
		}
	}
	return f
}

func upkeepIntent(v View) *Intent {
	f := v.Facts
	trip := f[FactTrip] > 0
	open := f[FactPlanMinOpen] > 0 && f[FactPlanBuy] > 0
	if !trip && !open {
		return nil
	}
	in := Intent{Kind: "upkeep", Class: Upkeep, Urgency: 0.5, NeedsArea: Town, EstCost: 60, Reason: "town trip running or chores due"}
	if open {
		in.Reason = fmt.Sprintf("plan would buy %d, a line is under its min", f[FactPlanBuy])
	}
	if f[FactTripEmerg] > 0 {
		in.Urgency, in.Reason = 0.9, "emergency chores"
		return &in
	}
	if f[FactPlanVerdict] == PlanShort && !trip {
		// short of gold: sell first when there is something to sell; otherwise it is recovery's business (as resupply was)
		if f[FactSellable] == 0 {
			return nil
		}
		in.Urgency, in.Steps = 0.85, []string{"sell_junk", "buy_potions"}
		in.Reason = fmt.Sprintf("short %d gold: sell %d items first", f[FactPlanShort], f[FactSellable])
	}
	if f[FactPlanVerdict] == PlanBlocked && !trip {
		return nil
	}
	return &in
}

// UpkeepSource is the one source that replaces SrcTrip and SrcResupply. It also answers a town visit so a visit that
// another upkeep intent started does the plan's buying in the same trip.
func UpkeepSource() Source {
	return FuncSource{N: SrcUpkeep, F: upkeepIntent, Town: func(v View) *Intent {
		if v.Facts[FactPlanBuy] > 0 {
			return upkeepIntent(v)
		}
		return nil
	}}
}

// UpkeepSources is StdSources with the trip and resupply sources collapsed into the one upkeep source.
func UpkeepSources() []Source {
	var out []Source
	for _, s := range StdSources() {
		switch s.Name() {
		case SrcTrip:
			out = append(out, UpkeepSource())
		case SrcResupply:
		default:
			out = append(out, s)
		}
	}
	return out
}
