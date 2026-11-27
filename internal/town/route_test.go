package town

import (
	"math"
	"slices"
	"testing"

	"d2lod-bot/internal/input"
)

// the fixed chore list of Chores with the stage 2 belt-first step, every chore enabled
var fixedList = []string{"idtome", "identify", "bulkid", "equip", "beltfirst", "sell", "ammo", "bosspurse", "beltbuy", "repair", "belt",
	"bank", "withdraw", "shop", "beltafter", "heal", "cube", "stash", "tidy"}

// two town layouts (an act 1 like one and a spread one): stop positions in subtiles, names are labels only.
var layouts = map[string]map[string]input.Point{
	"compact": {"repair": {X: 100, Y: 20}, "shop": {X: 20, Y: 20}, "cain": {X: 60, Y: 60}, "stash": {X: 60, Y: 20}},
	"spread":  {"repair": {X: 10, Y: 10}, "shop": {X: 200, Y: 10}, "cain": {X: 110, Y: 150}, "stash": {X: 110, Y: 10}},
}

func satisfies(order []string, edges [][2]string) bool {
	for _, e := range edges {
		if a, b := slices.Index(order, e[0]), slices.Index(order, e[1]); a < 0 || b < 0 || a > b {
			return false
		}
	}
	return true
}

func count(stops []string, s string) int {
	n := 0
	for _, x := range stops {
		if x == s {
			n++
		}
	}
	return n
}

func TestRouteOneVisitOneStashStop(t *testing.T) {
	for lname, pos := range layouts {
		for _, settleFirst := range []bool{false, true} {
			names := slices.Clone(fixedList)
			if !settleFirst {
				names = slices.DeleteFunc(names, func(n string) bool { return n == "bosspurse" }) // Chores plans it only when needed
			}
			edges := routeEdges(names, settleFirst)
			res := routeOrder(names, edges, pos, input.Point{X: 5, Y: 5})
			if len(res.Order) != len(names) || !satisfies(res.Order, edges) {
				t.Fatalf("%s settleFirst=%v: order %v breaks a precedence rule", lname, settleFirst, res.Order)
			}
			if n := count(res.Stops, "stash"); n != 1 {
				t.Errorf("%s settleFirst=%v: stash visited %d times, stops %v", lname, settleFirst, n, res.Stops)
			}
			if !settleFirst {
				if res.Stops[len(res.Stops)-1] != "stash" {
					t.Errorf("%s: purse can pay, the stash stop must be last: %v", lname, res.Stops)
				}
			} else if i := slices.Index(res.Order, "bank"); i > slices.Index(res.Order, "beltbuy") || i > slices.Index(res.Order, "shop") {
				t.Errorf("%s: purse cannot pay, the settle must come before the paid chores: %v", lname, res.Order)
			}
			_, fixedCost := walkOf(names, pos, input.Point{X: 5, Y: 5}, defaultRouteCosts)
			if res.Cost > fixedCost {
				t.Errorf("%s settleFirst=%v: route %d walks more than the fixed list %d", lname, settleFirst, res.Cost, fixedCost)
			}
		}
	}
}

func TestRouteBeatsFixedListWhereItCan(t *testing.T) {
	// the fixed list visits the stash for the boss purse, the bank and the final stash; the route does it once
	pos := layouts["spread"]
	names := slices.Clone(fixedList)
	fixedStops, fixedCost := walkOf(names, pos, input.Point{}, defaultRouteCosts)
	if count(fixedStops, "stash") < 3 {
		t.Fatalf("fixture: the fixed list should visit the stash 3 times, got %v", fixedStops)
	}
	res := routeOrder(names, routeEdges(names, true), pos, input.Point{})
	if res.Cost >= fixedCost {
		t.Errorf("route cost %d, fixed %d: no saving", res.Cost, fixedCost)
	}
}

func TestRouteDeterministicAndKeepsListWhenNothingToGain(t *testing.T) {
	pos := layouts["compact"]
	names := []string{"identify", "sell", "repair", "shop", "heal", "stash"}
	a := routeOrder(names, routeEdges(names, false), pos, input.Point{X: 1, Y: 1})
	for range 20 {
		if b := routeOrder(slices.Clone(names), routeEdges(names, false), pos, input.Point{X: 1, Y: 1}); !slices.Equal(a.Order, b.Order) || a.Cost != b.Cost {
			t.Fatalf("route differs between runs: %v vs %v", a, b)
		}
	}
	same := map[string]input.Point{"repair": {X: 7, Y: 7}, "shop": {X: 7, Y: 7}, "cain": {X: 7, Y: 7}, "stash": {X: 7, Y: 7}}
	if r := routeOrder(fixedList, routeEdges(fixedList, false), same, input.Point{X: 7, Y: 7}); !slices.Equal(r.Order, fixedList) && !satisfies(r.Order, routeEdges(fixedList, false)) {
		t.Errorf("flat layout changed a valid list: %v", r.Order)
	}
}

func TestRouteCycleKeepsTheList(t *testing.T) {
	names := []string{"sell", "shop"}
	r := routeOrder(names, [][2]string{{"sell", "shop"}, {"shop", "sell"}}, layouts["compact"], input.Point{})
	if !slices.Equal(r.Order, names) {
		t.Errorf("a cycle must keep the list, got %v", r.Order)
	}
}

func TestRouteSellBeforeSpendAndIdentifyFirst(t *testing.T) {
	for lname, pos := range layouts {
		for _, sf := range []bool{false, true} {
			names := slices.Clone(fixedList)
			res := routeOrder(names, routeEdges(names, sf), pos, input.Point{X: 50, Y: 50})
			ix := func(n string) int { return slices.Index(res.Order, n) }
			for _, p := range []string{"ammo", "beltbuy", "repair", "shop"} {
				if ix("sell") > ix(p) {
					t.Errorf("%s sf=%v: sell after %s: %v", lname, sf, p, res.Order)
				}
			}
			if ix("identify") > ix("sell") || ix("beltfirst") > ix("sell") || ix("belt") > ix("shop") || ix("shop") > ix("beltafter") {
				t.Errorf("%s sf=%v: a precedence rule broke: %v", lname, sf, res.Order)
			}
		}
	}
}

// A wall between two stops makes the path longer than the straight line, and the order follows the path: with straight
// lines the repair vendor is nearest to the start, with the detour priced the shop comes first.
func TestRouteFollowsPathCostNotStraightLine(t *testing.T) {
	start := input.Point{X: 0, Y: 0}
	pos := map[string]input.Point{"repair": {X: 10, Y: 0}, "shop": {X: 0, Y: 12}}
	names := []string{"repair", "shop"} // no edge between them
	straight := routeOrderWith(names, nil, pos, start, defaultRouteCosts)
	if !slices.Equal(straight.Order, []string{"repair", "shop"}) {
		t.Fatalf("straight line: %v", straight.Order)
	}
	// the way to the repair vendor winds around a wall: 40 subtiles for 10 in a line
	rc := defaultRouteCosts
	rc.Walk = func(a, b input.Point) int {
		if a == pos["repair"] || b == pos["repair"] {
			return 40
		}
		return int(math.Round(math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))))
	}
	got := routeOrderWith(names, nil, pos, start, rc)
	if !slices.Equal(got.Order, []string{"shop", "repair"}) {
		t.Errorf("path cost: %v, want the shop first", got.Order)
	}
	// the penalty is data: a huge one makes every order cost the same per stop, a zero one prices the walk alone
	rc.StopPenalty = 0
	if c := routeOrderWith(names, nil, pos, start, rc).Cost; c != 12+40 {
		t.Errorf("penalty 0: cost %d, want the walk alone 52", c)
	}
}
