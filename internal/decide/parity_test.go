package decide

import (
	"sort"
	"strings"
	"testing"
)

// legacyDefault is the golden table of DEFAULT-PARITY (spec 5.1, ADR 0016 gate "a rule file can express today's
// defaults"): for each of the 23 points the answer the legacy knob or code gives, per context of the point. It is
// written from the code and the knob defaults, NOT from the catalogue (V = read in the named function, U = the
// legacy code has no such path, so "do what the code does today" is the answer).
//
// split is non-empty when the legacy answer changes INSIDE a band (the band edges are the SME's and cannot hold the
// knob value); then the context is counted as a remaining mismatch, whatever the rule file says.
func legacyDefault(point string, f map[string]string) (ans, split string) {
	is := func(field, vals string) bool { return contains(strings.Split(vals, "|"), f[field]) }
	switch point {
	case "t0.retreat.drink", "t0.unique.fight", "t2.mercenary":
		// numeric axis with finer edges (catalogue v3): TestDefaultParityNumericPoints calls the legacy functions
		// over every integer value of the band, so no golden row is kept here
		return oracleAnswer, ""
	case "t0.retreat.lethal":
		return "avoid", "" // U: empty placeholder list, never fires today
	case "t0.area.toohard":
		// proven by TestDefaultParityTooHard against task.TowerLevelWhy over every clvl 1..40
		return oracleAnswer, ""
	case "t0.retreat.repeat":
		if is("retreats", "3+") {
			return "leave", "" // V brAttempts 3
		}
		return "stay", ""
	case "t0.leave.safe":
		return "stay", "" // U no code path
	case "t0.shrine":
		return "avoid", "" // U shrines are not handled
	case "t1.restock":
		if is("tripgold", "lt60|60-300") {
			return "no", "" // V TripBudgetSpent: tripGoldFloor 300 (tripMinGold 60)
		}
		return "yes", ""
	case "t1.return":
		return "yes", "" // V FieldGate with minFieldBeltPct 0 always says Go
	case "t1.keep.kind":
		return "keep", "" // U pickit decides inside a kind; the default keeps
	case "t1.town.trigger":
		// V TownTripTrigger (tripHPMin 1) and town NeedsTrip (tripPackUsedPct 80)
		if is("packused", "ge80") || is("belt", "low") {
			return "go", ""
		}
		return "continue", ""
	case "t1.tp.lost":
		if is("gold", "lt100") {
			return "walk", "" // V kitTPGold 100
		}
		return "buy", ""
	case "t1.stash.full":
		return "sell", "" // U the stash chore sells
	case "t1.repair":
		if is("durability", "gt30") {
			return "skip", "" // V repairBelowPct 30
		}
		return "repair", ""
	case "t1.identify":
		if is("itemclass", "higher") {
			return "scroll", "" // V idScrollWanted: set, rare, unique, crafted
		}
		return "leave", "" // V sellUnidentified
	case "t1.potions.buy":
		return "best", "" // V BuyKit buys the plan's preferred tier
	case "t1.gamble":
		return "save", "" // U no gamble path
	case "t1.quest.imbue":
		return "weapon", "" // U no code path reads the answer (TASK-197)
	case "t1.quest.cain":
		return "do", "" // U the campaign plan does the chain
	case "t2.repeat":
		return "again", "" // V the goal order count decides
	case "t2.failstreak":
		if is("hc", "hardcore") {
			return "stop", "" // spec 5.1; hardcore is unbuilt
		}
		if is("deaths", "4+") {
			return "stop", "" // gearDeathsHigh 4
		}
		return "continue", ""
	case "t2.slowrun":
		return "continue", "" // U no measure
	case "t2.bossunsafe":
		return "skip", "" // U
	case "t2.reset.game":
		return "restart", "" // V freshAllowed always ends in a restart after its wait
	}
	return "", "no golden row"
}

// oracleAnswer marks a point whose parity is proven by TestDefaultParityNumericPoints (legacy functions as oracle).
const oracleAnswer = "(oracle)"

// wantRemaining is the documented count of band-interior splits (docs/research/decision-point-store-notes.md).
// The ADR 0016 gate "a rule file can express today's defaults" closes only when it is 0.
// Catalogue v4 closed the last 72 (t0.area.toohard at clvl 14: the key now carries merc instead of levelgap).
const wantRemaining = 0

func TestDefaultParity(t *testing.T) {
	cat := mustCat(t)
	rf, err := ShippedRuleFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Points) != 25 {
		t.Fatalf("%d points, the golden table has 25", len(cat.Points))
	}
	var match, mismatch, split, oracle int
	perPoint := map[string]int{}
	for _, p := range cat.Points {
		ctxs, err := cat.Contexts(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, ctx := range ctxs {
			want, why := legacyDefault(p.ID, ctx.Fields)
			if want == "" {
				t.Fatalf("%s: %s", p.ID, why)
			}
			def, err := cat.Default(ctx)
			if err != nil {
				t.Fatal(err)
			}
			file, ok := rf.Lookup(ctx)
			if !ok {
				t.Errorf("%s: rule file has no row for %s", p.ID, ctx.Key)
				continue
			}
			if def != file {
				t.Errorf("%s: catalogue default %q and rule file %q disagree at %s", p.ID, def, file, ctx.Key)
			}
			switch {
			case want == oracleAnswer:
				oracle++
			case why != "":
				split++
				perPoint[p.ID]++
				if _, ok := p.answer(want); !ok {
					t.Errorf("%s: golden answer %q is not an answer", p.ID, want)
				}
			case file != want:
				mismatch++
				t.Errorf("%s: legacy %q, rule file %q at %s", p.ID, want, file, ctx.Key)
			default:
				match++
			}
		}
	}
	var ids []string
	for id := range perPoint {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		t.Logf("split contexts %-18s %d", id, perPoint[id])
	}
	t.Logf("parity: %d match, %d mismatch, %d band-interior splits (remaining gap), %d contexts proven by the legacy-function oracle", match, mismatch, split, oracle)
	if split != wantRemaining {
		t.Errorf("remaining splits = %d, the notes document %d: update both", split, wantRemaining)
	}
}

// The refused T0 points of the first slice, by witness: the states Track 1's grid tripped over.
func TestRefusedPointWitnessesAgree(t *testing.T) {
	cat := mustCat(t)
	rf, _ := ShippedRuleFile()
	ans := func(point string, cat_ map[string]string, num map[string]float64) string {
		o := baseObs(cat, "sorceress").with(cat_, num)
		ctx, err := cat.Build(point, o, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		a, ok := rf.Lookup(ctx)
		if !ok {
			t.Fatalf("%s: no row", ctx.Key)
		}
		return a
	}
	for _, tc := range []struct {
		name, point, want string
		cat               map[string]string
		num               map[string]float64
	}{
		{"30 percent, potion held: drink", "t0.retreat.drink", "drink", map[string]string{"kit": "OK", "held": "hp"}, map[string]float64{"life": 30}},
		{"90 percent: keep fighting", "t0.retreat.drink", "fight", map[string]string{"kit": "OK", "held": "hp"}, map[string]float64{"life": 90}},
		{"30 percent, no potion, no kit: run", "t0.retreat.drink", "run", map[string]string{"kit": "NONE", "held": "none"}, map[string]float64{"life": 30}},
		{"10 percent, no potion, no kit: leave the game", "t0.retreat.drink", "leave", map[string]string{"kit": "NONE", "held": "none"}, map[string]float64{"life": 10}},
		{"kit NONE still returns", "t1.return", "yes", map[string]string{"kit": "NONE", "areaclass": "field"}, nil},
		{"full belt, pack not full: continue", "t1.town.trigger", "continue", map[string]string{"belt": "full"}, map[string]float64{"packused": 10}},
		{"low belt: go", "t1.town.trigger", "go", map[string]string{"belt": "low"}, map[string]float64{"packused": 10}},
		{"150 gold: no restock (floor 300)", "t1.restock", "no", map[string]string{"belt": "low"}, map[string]float64{"tripgold": 150}},
		{"300 gold: restock", "t1.restock", "yes", map[string]string{"belt": "low"}, map[string]float64{"tripgold": 300}},
		{"clvl 10: skip", "t0.area.toohard", "skip", nil, map[string]float64{"clvl": 10}},
		{"clvl 16: go", "t0.area.toohard", "go", nil, map[string]float64{"clvl": 16}},
		{"champion pack at 25 percent: avoid", "t0.unique.fight", "avoid", nil, map[string]float64{"life": 20}},
		{"magic item, scroll held: leave", "t1.identify", "leave", map[string]string{"itemclass": "magic", "scroll": "yes"}, nil},
		{"rare item, no scroll: scroll", "t1.identify", "scroll", map[string]string{"itemclass": "higher", "scroll": "no"}, nil},
		{"charm kind: keep", "t1.keep.kind", "keep", map[string]string{"itemkind": "charm"}, nil},
		{"3 deaths: continue (gearDeathsHigh 4)", "t2.failstreak", "continue", nil, map[string]float64{"deaths": 3}},
		{"4 deaths: stop", "t2.failstreak", "stop", nil, map[string]float64{"deaths": 4}},
		{"reset game", "t2.reset.game", "restart", nil, nil},
	} {
		if got := ans(tc.point, tc.cat, tc.num); got != tc.want {
			t.Errorf("%s: %s = %q, want %q", tc.name, tc.point, got, tc.want)
		}
	}
}
