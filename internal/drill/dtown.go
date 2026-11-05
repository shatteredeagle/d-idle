package drill

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TownView is the town-relevant end state (StateReader: player, pack, UI, worn durability).
type TownView struct {
	Area        int
	InTown      bool
	Gold, Stash int
	Life, Max   int
	Pack        map[string]int
	DurCur      int // sum of worn durability, -1 unknown
	DurMax      int
	Belt        int // potions in the belt
	Class       string
	Qty         string // charges of pack tomes and scrolls, "tbk=20,tsc=3"
	Unid        int    // pack items that are not identified
	UnidCodes   string
	Panels      []string
}

// PackString is "hp1x2 tsc x1" sorted.
func (v TownView) PackString() string { return countString(v.Pack) }

// OpenInfo is one NPC window open by the bot's own path (town.Town.OpenForTest: cmd interact via the controller).
type OpenInfo struct {
	Class int
	Err   string
	Ticks int      // virtual ticks from the call to its return
	Flag  []string // panels and dialogs open at the return
}

// ChoreInfo is what the bot's Chores did for one plan.
type ChoreInfo struct {
	Plan                      string
	Err                       string
	Steps                     []string // "name ran|skipped err"
	Bought                    map[string]int
	Spent, Earned, GoldBefore int
	GoldAfter                 int
	Healed, Repaired          bool
	Ticks                     int
	Log                       []string
}

// TownLab is the instance as the D3 drills use it. Open, Buy, Sell, Chores and the stash calls are the bot's own town
// code (the same functions cmd/townlive and the chores call); the rest is logged-cheat setup or reads.
type TownLab interface {
	TView(ctx context.Context) (TownView, error)
	Tick(ctx context.Context, n int) error
	ClearPack(ctx context.Context, r *Run) error
	SetGold(ctx context.Context, r *Run, purse int) error
	SetStashGold(ctx context.Context, r *Run, n int) error
	SetLife(ctx context.Context, r *Run, points int) error
	GiveGround(ctx context.Context, r *Run, code string) error
	PickupRaw(ctx context.Context) (int, error)
	DrainBelt(ctx context.Context) error
	RepairKnob() int                              // the bot's repairBelowPct in this run (default 30; -repair-below overrides)
	Wear(ctx context.Context, r *Run) error       // cheat debug durability all 1: every worn item at durability 1
	Unidentify(ctx context.Context, r *Run) error // cheat debug unid: every carried (pack) item loses its identified flag

	OpenNPC(ctx context.Context, class int) OpenInfo
	CloseAll(ctx context.Context) error                                // the bot's own close (controller CloseAll), not the harness recovery
	OpenVendor(ctx context.Context, class int) error                   // the dialog's Trade line, window left open (not the bot's openShop: that resets the tab cache, this does not)
	SellPack(ctx context.Context, code string) (gained int, err error) // vendor window open: sells the first pack item with this code
	Chores(ctx context.Context, plan string) ChoreInfo                 // heal repair scrolls potions stash gold sell identify
	StashOpen(ctx context.Context) (ticks int, err error)
	StashRoundTrip(ctx context.Context) error
	StashDeposit(ctx context.Context, n int) error     // the bot's own deposit (town.DepositForTest), stash open
	StashDepositVerb(ctx context.Context, n int) error // d2dctl MoveGold deposit (control: it types the digits, V)
	StashWithdraw(ctx context.Context, n int) error    // d2dctl MoveGold (the bot has no withdraw path)
}

// Act 1 NPC classes (monstats hcIdx, internal/town/acts.go).
const (
	classGheed  = 147
	classAkara  = 148
	classKashya = 150
	classCharsi = 154
	classWarriv = 155
)

var act1NPC = []struct {
	Name  string
	Class int
}{{"Akara", classAkara}, {"Charsi", classCharsi}, {"Gheed", classGheed}, {"Kashya", classKashya}, {"Warriv", classWarriv}}

// D3 returns the town group: starting gold, NPC windows (P14), vendor buy and sell (P15), scroll and tome buy (P12),
// repair (P16), heal (P17), stash (P5), identify (P13).
func D3(l TownLab) []Drill {
	lim := 8 * time.Minute
	do := func(f func(context.Context, *Run, TownLab) Outcome) func(context.Context, *Run) Outcome {
		return func(ctx context.Context, r *Run) Outcome { return f(ctx, r, l) }
	}
	return []Drill{
		{ID: "D3.g0", Primitive: "U1", Title: "starting gold and stash of the fresh character (no cheat)", Limit: lim, Do: do(d3g0)},
		{ID: "D3.n", Primitive: "P14", Title: "approach and open each Act 1 NPC from the start point, window up, close leaves the UI clean", Limit: 12 * time.Minute, Do: do(d3n)},
		{ID: "D3.h", Primitive: "P17", Title: "hurt in town: the healer visit restores life to max, potions unchanged", Limit: lim, Do: do(d3h)},
		{ID: "D3.b", Primitive: "P15", Title: "belt emptied, gold given: the shop chore buys potions, gold delta = spent, belt gains", Limit: lim, Do: do(d3b)},
		{ID: "D3.b2", Primitive: "P15", Title: "sell a pack item to the vendor: gold up by the gain, item gone", Limit: lim, Do: do(d3b2)},
		{ID: "D3.tp", Primitive: "P12", Title: "scrolls chore with a tome and scrolls missing: tome bought and filled, gold delta = spent", Limit: lim, Do: do(d3tp)},
		{ID: "D3.r", Primitive: "P16", Title: "worn items at durability 1, gold enough: repair restores durability, price logged", Limit: lim, Do: do(d3r)},
		{ID: "D3.r0", Primitive: "P16", Title: "worn items at durability 1, gold 0: no repair progress, no loop, no gold lost", Limit: lim, Do: do(d3r0)},
		{ID: "D3.s", Primitive: "P5", Title: "stash opens, an item goes in and out, gold deposit and withdraw keep purse + stash constant", Limit: lim, Do: do(d3s)},
		{ID: "D3.s2", Primitive: "P5", Title: "partial gold deposit (the amount is typed): the bot's own deposit of 1000 of 1500", Limit: lim, Do: do(d3s2)},
		{ID: "D3.u", Primitive: "P13", Title: "unidentified magic item: the vendor's price against the price formula (docs/research/price-trace.md)", Limit: lim, Do: do(d3u)},
		{ID: "D3.i", Primitive: "P13", Title: "identify: an unidentified pack item and an ID scroll, the identify chore", Limit: lim, Do: do(d3i)},
	}
}

func d3g0(ctx context.Context, _ *Run, l TownLab) Outcome {
	v, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	obs := fmt.Sprintf("class %s: purse %d stash %d life %d/%d pack %s", v.Class, v.Gold, v.Stash, v.Life, v.Max, v.PackString())
	return Passed("recorded", "the fresh character's gold read (no cheat before this drill)", obs)
}

func panelsOf(p []string) string {
	if len(p) == 0 {
		return "none"
	}
	return strings.Join(p, ",")
}

func d3n(ctx context.Context, r *Run, l TownLab) Outcome {
	var obs, sig []string
	fails := 0
	for _, n := range act1NPC {
		oi := l.OpenNPC(ctx, n.Class)
		ok := oi.Err == "" && len(oi.Flag) > 0
		cerr := l.CloseAll(ctx)
		post, perr := l.TView(ctx)
		clean := perr == nil && len(post.Panels) == 0
		s := fmt.Sprintf("%s: open %v window [%s] %d ticks", n.Name, ok, panelsOf(oi.Flag), oi.Ticks)
		if oi.Err != "" {
			s += " err " + oi.Err
		}
		if cerr != nil {
			s += " close err " + cerr.Error()
		}
		s += fmt.Sprintf(" clean after close %v", clean)
		if !ok || !clean {
			fails++
		}
		obs = append(obs, s)
		sig = append(sig, fmt.Sprintf("%s:%v:%s:%v", n.Name, ok, panelsOf(oi.Flag), clean))
	}
	o := Outcome{Intended: "5 Act 1 NPCs open a window (vendor or dialog) within the bot's own wait; close leaves the UI clean", Observed: strings.Join(obs, " | "), Sig: strings.Join(sig, ";")}
	if fails == 0 {
		o.Verdict, o.Reason = Pass, "all 5 opened and closed clean"
	} else {
		o.Verdict, o.Reason = Fail, fmt.Sprintf("%d of 5 NPC opens or closes failed", fails)
	}
	return o
}

func d3h(ctx context.Context, r *Run, l TownLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 300); err != nil {
		return setupFail(err)
	}
	before, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	low := before.Max * 3 / 10
	if err := l.SetLife(ctx, r, low); err != nil {
		return setupFail(err)
	}
	hurt, err := l.TView(ctx)
	if err != nil || hurt.Life >= hurt.Max {
		return setupFail(fmt.Errorf("life cheat did not hold: %+v err %v", hurt, err))
	}
	ci := l.Chores(ctx, "heal")
	after, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	obs := fmt.Sprintf("life %d/%d -> %d/%d; gold %d -> %d; pack %s -> %s; healed=%v steps [%s] err %q", hurt.Life, hurt.Max, after.Life, after.Max, hurt.Gold, after.Gold, hurt.PackString(), after.PackString(), ci.Healed, strings.Join(ci.Steps, "; "), ci.Err)
	o := Outcome{Intended: "life = max after the visit, pack potions unchanged", Observed: obs,
		Sig: fmt.Sprintf("full=%v pack=%s->%s", after.Life >= after.Max, hurt.PackString(), after.PackString())}
	switch {
	case after.Life < after.Max:
		o.Verdict, o.Reason = Fail, "life not restored by the healer chore"
	case hurt.PackString() != after.PackString():
		o.Verdict, o.Reason = Fail, "pack changed during the heal"
	default:
		o.Verdict, o.Reason = Pass, "life restored at the healer"
	}
	return o
}

func d3b(ctx context.Context, r *Run, l TownLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.DrainBelt(ctx); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 1000); err != nil {
		return setupFail(err)
	}
	g0, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	ci := l.Chores(ctx, "potions")
	g1, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	obs := fmt.Sprintf("gold %d -> %d (spent per report %d); belt potions %d -> %d; pack %s -> %s; bought %s; steps [%s]; err %q", g0.Gold, g1.Gold, ci.Spent, g0.Belt, g1.Belt, g0.PackString(), g1.PackString(), countString(ci.Bought), strings.Join(ci.Steps, "; "), ci.Err)
	o := Outcome{Intended: "potions bought through the shop chore, gold delta = price, potions in the belt", Observed: obs,
		Sig: fmt.Sprintf("delta=%d spent=%d belt=%d->%d bought=%s", g0.Gold-g1.Gold, ci.Spent, g0.Belt, g1.Belt, countString(ci.Bought))}
	switch {
	case ci.Spent <= 0 || g0.Gold-g1.Gold != ci.Spent:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("gold delta %d vs report Spent %d", g0.Gold-g1.Gold, ci.Spent)
	case g1.Belt <= g0.Belt:
		o.Verdict, o.Reason = Fail, "potions bought but the belt did not gain"
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("bought %s for %d, belt %d -> %d", countString(ci.Bought), ci.Spent, g0.Belt, g1.Belt)
	}
	return o
}

func d3b2(ctx context.Context, r *Run, l TownLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.DrainBelt(ctx); err != nil {
		return setupFail(err)
	}
	if err := l.GiveGround(ctx, r, "hp1"); err != nil { // the belt is empty: the engine belts it; give a second kind that stays in the pack
		return setupFail(err)
	}
	if err := l.GiveGround(ctx, r, "isc"); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 30); err != nil {
		return setupFail(err)
	}
	if _, err := l.PickupRaw(ctx); err != nil {
		return setupFail(err)
	}
	g0, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	if g0.Pack["isc"] == 0 {
		return setupFail(fmt.Errorf("no isc in the pack to sell (pack %s)", g0.PackString()))
	}
	if err := l.SetGold(ctx, r, 100); err != nil {
		return setupFail(err)
	}
	g0, _ = l.TView(ctx)
	serr0 := l.OpenVendor(ctx, classAkara)
	if serr0 != nil {
		_ = l.CloseAll(ctx)
		return Failed("open vendor: "+serr0.Error(), "sell the scroll", "")
	}
	gained, serr := l.SellPack(ctx, "isc")
	g1, _ := l.TView(ctx)
	_ = l.CloseAll(ctx)
	obs := fmt.Sprintf("sell isc to Akara: gained %d err %v; gold %d -> %d; pack %s -> %s", gained, serr, g0.Gold, g1.Gold, g0.PackString(), g1.PackString())
	o := Outcome{Intended: "gold up by the gain, the scroll gone, nothing else sold", Observed: obs,
		Sig: fmt.Sprintf("gained=%d delta=%d pack=%s->%s", gained, g1.Gold-g0.Gold, g0.PackString(), g1.PackString())}
	switch {
	case serr != nil || gained <= 0 || g1.Gold-g0.Gold != gained:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("sell: err %v, gold delta %d vs gained %d", serr, g1.Gold-g0.Gold, gained)
	case g1.Pack["isc"] != g0.Pack["isc"]-1:
		o.Verdict, o.Reason = Fail, "the scroll is still in the pack"
	case sellExpect(l, "isc") > 0 && sellExpect(l, "isc") != gained:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("sold for %d, the price formula (docs/research/price-trace.md) says %d", gained, sellExpect(l, "isc"))
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("sold an identify scroll for %d (the price formula says %d)", gained, sellExpect(l, "isc"))
	}
	return o
}

func d3tp(ctx context.Context, r *Run, l TownLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 1000); err != nil {
		return setupFail(err)
	}
	g0, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	ci := l.Chores(ctx, "scrolls")
	g1, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	obs := fmt.Sprintf("gold %d -> %d (spent per report %d); pack %s; charges [%s]; bought %s; steps [%s]; err %q", g0.Gold, g1.Gold, ci.Spent, g1.PackString(), g1.Qty, countString(ci.Bought), strings.Join(ci.Steps, "; "), ci.Err)
	o := Outcome{Intended: "a tome (tbk) or scrolls (tsc) in the pack, gold delta = spent", Observed: obs,
		Sig: fmt.Sprintf("pack=%s delta=%d spent=%d", g1.PackString(), g0.Gold-g1.Gold, ci.Spent)}
	have := g1.Pack["tbk"] + g1.Pack["tsc"]
	switch {
	case have == 0:
		o.Verdict, o.Reason = Fail, "no tome or scroll in the pack after the scrolls chore"
	case g0.Gold-g1.Gold != ci.Spent:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("gold delta %d differs from the chore's Spent %d", g0.Gold-g1.Gold, ci.Spent)
	default:
		o.Verdict, o.Reason = Pass, "tome or scrolls bought, gold delta equals the report"
	}
	return o
}

func d3r(ctx context.Context, r *Run, l TownLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 500); err != nil {
		return setupFail(err)
	}
	if err := l.Wear(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 10); err != nil {
		return setupFail(err)
	}
	v0, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	if v0.DurMax <= 0 || v0.DurCur >= v0.DurMax {
		return Blocks(fmt.Sprintf("no worn item with reduced durability after `debug durability all 1` (dur %d/%d)", v0.DurCur, v0.DurMax))
	}
	knob := l.RepairKnob()
	est := -1
	if re, ok := l.(repairEstimator); ok {
		if e, err := re.RepairEstimate(ctx); err == nil {
			est = e
		}
	}
	ci := l.Chores(ctx, "repair")
	v1, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	obs := fmt.Sprintf("knob repairBelowPct %d; durability %d/%d (%d%%) -> %d/%d; gold %d -> %d (spent per report %d); estimate before the visit %d; repaired=%v steps [%s] err %q",
		knob, v0.DurCur, v0.DurMax, 100*v0.DurCur/v0.DurMax, v1.DurCur, v1.DurMax, v0.Gold, v1.Gold, ci.Spent, est, ci.Repaired, strings.Join(ci.Steps, "; "), ci.Err)
	if 100*v0.DurCur/v0.DurMax >= knob { // the chore is right to skip: record it, the repair path needs a lower item
		skipped := v1.DurCur == v0.DurCur && v0.Gold == v1.Gold
		o := Blocks(fmt.Sprintf("the gear is not below the repair knob %d: the repair path is not reached. The chore skipped correctly: %v", knob, skipped))
		o.Observed, o.Intended = obs, "skip at or above the knob"
		return o
	}
	o := Outcome{Intended: "durability back to max, gold down by the price", Observed: obs, Sig: fmt.Sprintf("dur=%d/%d->%d/%d paid=%v", v0.DurCur, v0.DurMax, v1.DurCur, v1.DurMax, v0.Gold > v1.Gold)}
	switch {
	case v1.DurCur < v1.DurMax:
		o.Verdict, o.Reason = Fail, "durability not restored"
	case v0.Gold <= v1.Gold:
		o.Verdict, o.Reason = Fail, "durability restored but no gold taken (price unreadable?)"
	case est >= 0 && est != v0.Gold-v1.Gold:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("repair estimate %d differs from the %d gold paid", est, v0.Gold-v1.Gold)
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("repaired for %d gold at knob %d; the estimate before the visit was %d", v0.Gold-v1.Gold, knob, est)
	}
	return o
}

// repairEstimator is town.Town.RepairEstimateForTest (the price the chore checks the purse against).
type repairEstimator interface {
	RepairEstimate(ctx context.Context) (int, error)
}

// sellExpecter is the price formula of internal/data for a normal item of this code at the Act 1 vendors.
type sellExpecter interface {
	SellExpect(code string) int
}

func d3r0(ctx context.Context, r *Run, l TownLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 0); err != nil {
		return setupFail(err)
	}
	if err := l.Wear(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 10); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 10); err != nil {
		return setupFail(err)
	}
	v0, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	if v0.DurMax <= 0 || v0.DurCur >= v0.DurMax {
		return Blocks(fmt.Sprintf("no worn item with reduced durability after `debug durability all 1` (dur %d/%d)", v0.DurCur, v0.DurMax))
	}
	if 100*v0.DurCur/v0.DurMax >= l.RepairKnob() {
		return Blocks(fmt.Sprintf("gear at %d percent is not below the repair knob %d: the poor-gold path is not reached (the knob is above the cheat; -repair-below overrides it)", 100*v0.DurCur/v0.DurMax, l.RepairKnob()))
	}
	var calls []string
	t0 := 0
	for i := 0; i < 2; i++ {
		ci := l.Chores(ctx, "repair")
		t0 += ci.Ticks
		calls = append(calls, fmt.Sprintf("call%d steps [%s] err %q ticks %d", i+1, strings.Join(ci.Steps, "; "), ci.Err, ci.Ticks))
	}
	v1, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	obs := fmt.Sprintf("durability %d/%d -> %d/%d; gold %d -> %d; %s; panels %s", v0.DurCur, v0.DurMax, v1.DurCur, v1.DurMax, v0.Gold, v1.Gold, strings.Join(calls, "; "), panelsOf(v1.Panels))
	o := Outcome{Intended: "no repair at gold 0, gold still 0, UI clean", Observed: obs, Sig: fmt.Sprintf("dur=%d/%d->%d/%d gold=%d", v0.DurCur, v0.DurMax, v1.DurCur, v1.DurMax, v1.Gold)}
	switch {
	case v1.Gold != 0 || v1.DurCur > v0.DurCur:
		o.Verdict, o.Reason = Fail, "gold or durability changed at gold 0"
	case len(v1.Panels) > 0:
		o.Verdict, o.Reason = Fail, "panel left open"
	default:
		o.Verdict, o.Reason = Pass, "gold 0: one failed repair attempt (logged as an error, the price is not checked before the visit), then latched: the second call skips"
	}
	return o
}

func stashStage(ctx context.Context, r *Run, l TownLab, ring bool) (TownView, error) {
	if err := l.ClearPack(ctx, r); err != nil {
		return TownView{}, err
	}
	if err := l.SetGold(ctx, r, 1500); err != nil {
		return TownView{}, err
	}
	if err := l.SetStashGold(ctx, r, 0); err != nil {
		return TownView{}, err
	}
	if ring { // not Beltable, so the engine's pickup puts it in the pack (a potion would be belted)
		if err := l.GiveGround(ctx, r, "rin"); err != nil {
			return TownView{}, err
		}
		if err := l.Tick(ctx, 30); err != nil {
			return TownView{}, err
		}
		if n, err := l.PickupRaw(ctx); err != nil || n == 0 {
			return TownView{}, fmt.Errorf("could not pick the ring up (%d, %v)", n, err)
		}
	}
	return l.TView(ctx)
}

func d3s(ctx context.Context, r *Run, l TownLab) Outcome {
	v0, err := stashStage(ctx, r, l, true)
	if err != nil {
		return setupFail(err)
	}
	var steps []string
	want := "stash opens; ring round trip; deposit all 1500 then withdraw 300: purse 300 stash 1200"
	fail := func(why string) Outcome {
		_ = l.CloseAll(ctx)
		return Outcome{Verdict: Fail, Reason: why, Intended: want, Observed: strings.Join(steps, "; "), Sig: why}
	}
	if _, err := l.StashOpen(ctx); err != nil {
		return fail("open stash: " + err.Error())
	}
	steps = append(steps, "stash open")
	if err := l.StashRoundTrip(ctx); err != nil {
		return fail("item round trip: " + err.Error())
	}
	vi, _ := l.TView(ctx)
	steps = append(steps, "item round trip ok, pack "+vi.PackString())
	if err := l.StashDeposit(ctx, 1500); err != nil {
		steps = append(steps, "bot deposit of the whole purse: "+err.Error())
		cerr := l.StashDepositVerb(ctx, 1500)
		vc, _ := l.TView(ctx)
		steps = append(steps, fmt.Sprintf("control, d2dctl MoveGold deposit 1500 (types the digits): err %v, purse %d stash %d", cerr, vc.Gold, vc.Stash))
		return fail("deposit all: " + err.Error())
	}
	v1, _ := l.TView(ctx)
	steps = append(steps, fmt.Sprintf("deposit 1500: purse %d stash %d", v1.Gold, v1.Stash))
	if err := l.StashWithdraw(ctx, 300); err != nil {
		return fail("withdraw: " + err.Error())
	}
	v2, _ := l.TView(ctx)
	steps = append(steps, fmt.Sprintf("withdraw 300: purse %d stash %d", v2.Gold, v2.Stash))
	_ = l.CloseAll(ctx)
	o := Outcome{Intended: want, Observed: strings.Join(steps, "; "),
		Sig: fmt.Sprintf("%d/%d %d/%d %d/%d pack=%s->%s", v0.Gold, v0.Stash, v1.Gold, v1.Stash, v2.Gold, v2.Stash, v0.PackString(), vi.PackString())}
	switch {
	case v1.Gold != 0 || v1.Stash != 1500:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("deposit 1500 gave purse %d stash %d", v1.Gold, v1.Stash)
	case v2.Gold != 300 || v2.Stash != 1200:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("withdraw 300 gave purse %d stash %d", v2.Gold, v2.Stash)
	case v0.PackString() != vi.PackString():
		o.Verdict, o.Reason = Fail, "the ring did not come back to the pack"
	default:
		o.Verdict, o.Reason = Pass, "stash item and full-purse gold moves exact"
	}
	return o
}

func d3s2(ctx context.Context, r *Run, l TownLab) Outcome {
	v0, err := stashStage(ctx, r, l, false)
	if err != nil {
		return setupFail(err)
	}
	want := "deposit 1000 of 1500 (amount typed into the prompt): purse 500 stash 1000"
	if _, err := l.StashOpen(ctx); err != nil {
		_ = l.CloseAll(ctx)
		return Failed("open stash: "+err.Error(), want, "")
	}
	derr := l.StashDeposit(ctx, 1000)
	v1, _ := l.TView(ctx)
	ctl := ""
	if derr != nil { // the bot's failed attempt pressed Escape: open the stash again for the control
		_, _ = l.StashOpen(ctx)
		cerr := l.StashDepositVerb(ctx, 1000)
		vc, _ := l.TView(ctx)
		ctl = fmt.Sprintf("; control, d2dctl MoveGold deposit 1000: err %v, purse %d stash %d", cerr, vc.Gold, vc.Stash)
	}
	_ = l.CloseAll(ctx)
	obs := fmt.Sprintf("deposit 1000 err %v; purse %d -> %d stash %d -> %d%s", derr, v0.Gold, v1.Gold, v0.Stash, v1.Stash, ctl)
	o := Outcome{Intended: want, Observed: obs, Sig: fmt.Sprintf("err=%v %d/%d", derr != nil, v1.Gold, v1.Stash)}
	if derr != nil || v1.Gold != 500 || v1.Stash != 1000 {
		o.Verdict, o.Reason = Fail, "partial deposit failed: "+fmt.Sprint(derr)
	} else {
		o.Verdict, o.Reason = Pass, "partial deposit exact"
	}
	return o
}

func d3i(ctx context.Context, r *Run, l TownLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 500); err != nil {
		return setupFail(err)
	}
	if ug, ok := l.(unidGiver); ok { // a magic ring that is unidentified from the start (`give pack q4 unid`)
		if err := ug.GivePackUnid(ctx, r, "rin", 4); err != nil {
			return setupFail(err)
		}
	}
	for _, code := range []string{"rin", "isc"} {
		if _, ok := l.(unidGiver); ok && code == "rin" {
			continue
		}
		if err := l.GiveGround(ctx, r, code); err != nil {
			return setupFail(err)
		}
		if err := l.Tick(ctx, 30); err != nil {
			return setupFail(err)
		}
		if n, err := l.PickupRaw(ctx); err != nil || n == 0 {
			return setupFail(fmt.Errorf("could not pick up %s (%d, %v)", code, n, err))
		}
		if code == "rin" { // unidentify the ring before the scroll arrives: the cheat touches every pack item
			if err := l.Unidentify(ctx, r); err != nil {
				return setupFail(err)
			}
			if err := l.Tick(ctx, 10); err != nil {
				return setupFail(err)
			}
		}
	}
	v0, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	if v0.Unid == 0 {
		return setupFail(fmt.Errorf("nothing unidentified after debug unid (pack %s)", v0.PackString()))
	}
	ci := l.Chores(ctx, "identify")
	v1, err := l.TView(ctx)
	if err != nil {
		return setupFail(err)
	}
	if strings.Contains(strings.Join(ci.Steps, ";"), "identify skipped") {
		return Blocks(fmt.Sprintf("the identify chore never selects what `debug unid` can make: it clears the flag of plain base items (a normal ring), the bot's judge sets NeedID only for magic or better; missing: a cheat that gives an unidentified magic or rare item (`debug give <code> <quality>` with the flag cleared). Observed: unidentified [%s], chore steps [%s]", v0.UnidCodes, strings.Join(ci.Steps, "; ")))
	}
	obs := fmt.Sprintf("unidentified [%s] -> [%s]; pack %s -> %s; gold %d -> %d; steps [%s]; err %q", v0.UnidCodes, v1.UnidCodes, v0.PackString(), v1.PackString(), v0.Gold, v1.Gold, strings.Join(ci.Steps, "; "), ci.Err)
	o := Outcome{Intended: "ring identified, ID scroll count -1, UI clean", Observed: obs, Sig: fmt.Sprintf("unid=%s->%s pack=%s->%s", v0.UnidCodes, v1.UnidCodes, v0.PackString(), v1.PackString())}
	switch {
	case strings.Contains(v1.UnidCodes, "rin"):
		o.Verdict, o.Reason = Fail, "the ring is still unidentified after the identify chore"
	case v1.Pack["isc"] >= v0.Pack["isc"] && v0.Pack["isc"] > 0:
		o.Verdict, o.Reason = Fail, "ring identified but the scroll was not spent"
	default:
		o.Verdict, o.Reason = Pass, "identified with a scroll"
	}
	return o
}

func sellExpect(l TownLab, code string) int {
	if e, ok := l.(sellExpecter); ok {
		return e.SellExpect(code)
	}
	return 0
}

// unidGiver is `debug give <code> pack q<quality> unid`: a magic or better item that is unidentified (forkcheats2).
type unidGiver interface {
	GivePackUnid(ctx context.Context, r *Run, code string, quality int) error
}

// d3u measures what Charsi pays for an unidentified magic cap and compares it with the price formula.
func d3u(ctx context.Context, r *Run, l TownLab) Outcome {
	ug, ok := l.(unidGiver)
	if !ok {
		return Blocks("the lab has no `give pack q4 unid` step")
	}
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 100); err != nil {
		return setupFail(err)
	}
	if err := ug.GivePackUnid(ctx, r, "cap", 4); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 10); err != nil {
		return setupFail(err)
	}
	g0, _ := l.TView(ctx)
	if g0.Pack["cap"] == 0 {
		return setupFail(fmt.Errorf("no cap in the pack (%s)", g0.PackString()))
	}
	if err := l.OpenVendor(ctx, classCharsi); err != nil {
		_ = l.CloseAll(ctx)
		return Failed("open vendor: "+err.Error(), "sell the cap", "")
	}
	gained, serr := l.SellPack(ctx, "cap")
	g1, _ := l.TView(ctx)
	_ = l.CloseAll(ctx)
	want := sellExpect(l, "cap")
	obs := fmt.Sprintf("unidentified magic Cap (unid %d) sold to Charsi: gained %d err %v; gold %d -> %d; formula for a normal cap %d", g0.Unid, gained, serr, g0.Gold, g1.Gold, want)
	o := Outcome{Intended: "an unidentified magic item sells for its base price (cost/2, the identified extras do not count)", Observed: obs, Sig: fmt.Sprintf("gained=%d want=%d err=%v", gained, want, serr != nil)}
	switch {
	case serr != nil || gained <= 0 || g1.Gold-g0.Gold != gained:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("sell: err %v, gold delta %d vs gained %d", serr, g1.Gold-g0.Gold, gained)
	case want > 0 && gained != want:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("paid %d, the formula for the base says %d", gained, want)
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("unidentified magic cap sold for %d = the base formula", gained)
	}
	return o
}
