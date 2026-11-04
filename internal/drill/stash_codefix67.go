package drill

import (
	"context"
	"fmt"
	"time"
)

// Purses of the P5 stash recheck (docs/codefix67.md "Live recheck recipe", docs/livecheckdrills.md). Rich must pay the
// whole kit; Poor must be below the keep line and below the kit.
const (
	stashRich  = 600
	stashPoor  = 6
	stashStock = 1400
)

// stashPlan: the chore step `withdraw` runs only with plan.Potions && plan.Gold (internal/town/chores.go), `stash` with Stash or Gold.
const stashPlan = "potions gold stash"

// StashLab is the optional part of the lab for the stash recheck. All numbers come from the bot's tuning, not from here.
type StashLab interface {
	TownLab
	KeepGold() int                                           // the purse the withdrawal tops up to (town keep line)
	KitCost() int                                            // resource.Tuning kit cost: KitTPGold + max(HPFirst,1)*KitPotionGold
	RejuvCode() string                                       // item code of the smallest rejuvenation potion (from the class tables)
	GivePack(ctx context.Context, r *Run, code string) error // cheat `debug give <code> pack`: the item lands in the pack, never the belt
}

// TripLab is what scenario 3 needs and no lab has yet: run the bot's own Runner loop in town for a few rounds and
// count its `town trip ... emergency=true` log lines and stash rounds (internal/bot goal.go logs the line; the latch is
// Runner state, so TownLab.Chores cannot show it).
type TripLab interface {
	RunnerRounds(ctx context.Context, rounds int) (emergencyTrips, stashRounds int, err error)
}

// DStash returns the P5 stash recheck: DS.1 purse pays the kit, DS.2 purse 6 withdraws once, DS.3 futile round latch,
// DS.4 one rejuvenation in the pack means no withdrawal.
func DStash(l GameLab, lim time.Duration) []Drill {
	sl, ok := l.(StashLab)
	if !ok {
		return []Drill{{ID: "DS", Primitive: "P5", Title: "stash recheck of codefix67", Needs: []string{"the lab has no KeepGold, KitCost, RejuvCode"}}}
	}
	do := func(f func(context.Context, *Run, StashLab) Outcome) func(context.Context, *Run) Outcome {
		return func(ctx context.Context, r *Run) Outcome { return f(ctx, r, sl) }
	}
	ds3 := Drill{ID: "DS.3", Primitive: "P5", Title: "6 gold, stash empty, shop cannot sell: one round, then no further emergency trip line", Limit: lim}
	if tl, ok := l.(TripLab); ok {
		ds3.Do = func(ctx context.Context, r *Run) Outcome { return ds3Run(ctx, r, sl, tl) }
	} else {
		ds3.Needs = []string{"TripLab.RunnerRounds: the bot Runner loop in town (the futility latch is Runner state; the lab only runs Chores), plus a way to make the shop unable to sell (no d2d verb empties a vendor stock)"}
	}
	return []Drill{
		{ID: "DS.1", Primitive: "P5", Title: "600 purse, 1400 stash: no stash gold moves, the shop sells the kit", Limit: lim, Do: do(ds1)},
		{ID: "DS.2", Primitive: "P5", Title: "6 purse, 1400 stash: one withdrawal up to the keep line, then potions in the belt", Limit: lim, Do: do(ds2)},
		ds3,
		{ID: "DS.4", Primitive: "P5", Title: "one rejuvenation in the pack, 6 purse, 1400 stash: no withdrawal", Limit: lim, Do: do(ds4)},
	}
}

func stashSetup(ctx context.Context, r *Run, l StashLab, purse, stash int) error {
	if err := l.ClearPack(ctx, r); err != nil {
		return err
	}
	if err := l.DrainBelt(ctx); err != nil {
		return err
	}
	if err := l.SetGold(ctx, r, purse); err != nil {
		return err
	}
	return l.SetStashGold(ctx, r, stash)
}

// stashRun runs the chores once and reads the town view before and after.
func stashRun(ctx context.Context, l StashLab) (g0, g1 TownView, ci ChoreInfo, err error) {
	if g0, err = l.TView(ctx); err != nil {
		return
	}
	ci = l.Chores(ctx, stashPlan)
	g1, err = l.TView(ctx)
	return
}

func stashObs(g0, g1 TownView, ci ChoreInfo) string {
	return fmt.Sprintf("purse %d -> %d, stash %d -> %d, belt %d -> %d, spent %d, bought %s, err %q", g0.Gold, g1.Gold, g0.Stash, g1.Stash, g0.Belt, g1.Belt, ci.Spent, countString(ci.Bought), ci.Err)
}

func ds1(ctx context.Context, r *Run, l StashLab) Outcome {
	if k := l.KitCost(); k <= 0 || k > stashRich {
		return Blocks(fmt.Sprintf("kit cost %d is not within the %d-gold purse of the scenario", k, stashRich))
	}
	if err := stashSetup(ctx, r, l, stashRich, stashStock); err != nil {
		return setupFail(err)
	}
	g0, g1, ci, err := stashRun(ctx, l)
	if err != nil {
		return setupFail(err)
	}
	o := Outcome{Intended: fmt.Sprintf("stash stays %d (no withdrawal), spent > 0, belt gains", stashStock), Observed: stashObs(g0, g1, ci),
		Sig: fmt.Sprintf("stash=%d->%d spent=%v belt=%v", g0.Stash, g1.Stash, ci.Spent > 0, g1.Belt > g0.Belt)}
	switch {
	case g1.Stash < g0.Stash:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("stash gold fell by %d although the purse paid the kit", g0.Stash-g1.Stash)
	case ci.Spent <= 0 || g1.Belt <= g0.Belt:
		o.Verdict, o.Reason = Fail, "no shop purchase (spent or belt did not grow)"
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("no withdrawal, bought for %d", ci.Spent)
	}
	return o
}

func ds2(ctx context.Context, r *Run, l StashLab) Outcome {
	keep := l.KeepGold()
	if keep <= stashPoor || keep-stashPoor > stashStock || l.KitCost() <= stashPoor {
		return Blocks(fmt.Sprintf("keep line %d / kit %d do not fit the purse %d and stash %d of the scenario", keep, l.KitCost(), stashPoor, stashStock))
	}
	if err := stashSetup(ctx, r, l, stashPoor, stashStock); err != nil {
		return setupFail(err)
	}
	g0, g1, ci, err := stashRun(ctx, l)
	if err != nil {
		return setupFail(err)
	}
	drawn := g0.Stash - g1.Stash
	o := Outcome{Intended: fmt.Sprintf("stash falls by keep line %d minus purse %d = %d, belt gains", keep, stashPoor, keep-stashPoor), Observed: stashObs(g0, g1, ci),
		Sig: fmt.Sprintf("drawn=%d belt=%v", drawn, g1.Belt > g0.Belt)}
	switch {
	case drawn != keep-stashPoor:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("withdrew %d, wanted %d", drawn, keep-stashPoor)
	case g1.Belt <= g0.Belt:
		o.Verdict, o.Reason = Fail, "withdrew but no potions reached the belt"
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("withdrew %d once, belt %d -> %d", drawn, g0.Belt, g1.Belt)
	}
	return o
}

func ds3Run(ctx context.Context, r *Run, l StashLab, tl TripLab) Outcome {
	if err := stashSetup(ctx, r, l, stashPoor, 0); err != nil {
		return setupFail(err)
	}
	const rounds = 4
	trips, rounds1, err := tl.RunnerRounds(ctx, rounds)
	if err != nil {
		return setupFail(err)
	}
	o := Outcome{Intended: "at most one stash round and one emergency trip line while gold stays the same", Observed: fmt.Sprintf("%d rounds: %d emergency trip lines, %d stash rounds", rounds, trips, rounds1),
		Sig: fmt.Sprintf("trips<=1=%v rounds<=1=%v", trips <= 1, rounds1 <= 1)}
	if trips > 1 || rounds1 > 1 {
		o.Verdict, o.Reason = Fail, "the futile in-town round repeated (latch not held)"
		return o
	}
	o.Verdict, o.Reason = Pass, "latch held"
	return o
}

func ds4(ctx context.Context, r *Run, l StashLab) Outcome {
	code := l.RejuvCode()
	if code == "" {
		return Blocks("no rejuvenation code in the class tables")
	}
	if err := stashSetup(ctx, r, l, stashPoor, stashStock); err != nil {
		return setupFail(err)
	}
	// `debug give <code>` on the ground and a pickup put a potion into the belt (live DS.4 left the pack empty)
	if err := l.GivePack(ctx, r, code); err != nil {
		return setupFail(err)
	}
	g0, g1, ci, err := stashRun(ctx, l)
	if err != nil {
		return setupFail(err)
	}
	o := Outcome{Intended: "rejuvenation in the pack, stash unchanged", Observed: stashObs(g0, g1, ci) + " pack " + g0.PackString(),
		Sig: fmt.Sprintf("stash=%d->%d", g0.Stash, g1.Stash)}
	switch {
	case g0.Pack[code] < 1:
		o.Verdict, o.Reason = Blocked, "the rejuvenation did not reach the pack: setup failed"
	case g1.Stash != g0.Stash:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("withdrew %d with a rejuvenation in the pack", g0.Stash-g1.Stash)
	default:
		o.Verdict, o.Reason = Pass, "no withdrawal"
	}
	return o
}
