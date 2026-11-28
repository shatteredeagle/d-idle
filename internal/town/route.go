package town

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Stage 5 of the town planner (knob town.plan >= 5, docs/design/fable-review-2026-11-21/town-planner.md section 4.3):
// the chores of a visit are ordered by the walk they cost, under precedence rules, instead of the fixed list. The pure
// part is here (routeOrder); Chores supplies the stop positions (NPC and object presets of the act table, read at run time).

// choreRole is where a chore is done: an NPC role of the act table, the stash, or "" for a chore that works from the
// inventory and walks nowhere. A chore absent from the table keeps its place at "" (never reordered across a rule it
// does not know, because it has no edges).
var choreRole = map[string]string{
	"idtome": "shop", "identify": "cain", "bulkid": "", "equip": "", "beltfirst": "",
	"sell": "repair", "ammo": "repair", "beltbuy": "repair", "repair": "repair",
	"belt": "", "bosspurse": "stash", "bank": "stash", "withdraw": "stash",
	"shop": "shop", "beltafter": "", "heal": "shop", "cube": "stash", "stash": "stash", "tidy": "stash",
}

// fixedEdges are the precedence rules that hold in every visit ("a before b"), each with its reason.
var fixedEdges = [][2]string{
	{"idtome", "identify"}, {"idtome", "sell"}, {"identify", "sell"}, {"bulkid", "sell"}, // identify before sell
	{"equip", "sell"}, {"beltfirst", "sell"}, // the belt and the body take their potions and gear before the sale sees them
	{"sell", "ammo"}, {"sell", "beltbuy"}, {"sell", "repair"}, {"sell", "shop"}, // sell before buy
	{"sell", "cube"}, {"sell", "stash"}, {"sell", "tidy"}, // stash what is left after the sale
	{"bosspurse", "beltbuy"},                                  // the boss purse is withdrawn before the belt buy keeps its reserve
	{"belt", "shop"}, {"shop", "beltafter"}, {"shop", "heal"}, // the shop buys only the shortfall; heal rides the potion stop
}

var (
	paidChores   = []string{"ammo", "beltbuy", "repair", "shop"} // gold is spent in this relative order, as the fixed list did
	settleChores = []string{"bosspurse", "bank", "withdraw"}     // the purse is settled at the stash
	itemChores   = []string{"cube", "stash", "tidy"}             // items go to the stash after the settle
)

// routeEdges returns the precedence pairs among the chores present. settleFirst: the purse cannot pay the plan, so the
// settle stop comes before the first paid chore; else the stash stop is last (design 4.3), which is one stash stop.
func routeEdges(names []string, settleFirst bool) [][2]string {
	has := map[string]bool{}
	for _, n := range names {
		has[n] = true
	}
	var out [][2]string
	add := func(a, b string) {
		if has[a] && has[b] {
			out = append(out, [2]string{a, b})
		}
	}
	for _, e := range fixedEdges {
		add(e[0], e[1])
	}
	chain := func(set []string) {
		var prev string
		for _, n := range set {
			if !has[n] {
				continue
			}
			if prev != "" {
				add(prev, n)
			}
			prev = n
		}
	}
	chain(paidChores)
	chain(append(slices.Clone(settleChores), itemChores...))
	if settleFirst {
		for _, s := range settleChores {
			add("sell", s) // one stash stop: the sale is behind us when the settle and the items share it
			for _, p := range paidChores {
				add(s, p)
			}
		}
		return out
	}
	for _, n := range names {
		if slices.Contains(settleChores, n) || slices.Contains(itemChores, n) {
			continue
		}
		for _, last := range append(slices.Clone(settleChores), itemChores...) {
			add(n, last)
		}
	}
	return out
}

// routeCosts is the price list of a walk, in subtiles. StopPenalty is what opening one more stop costs besides the walk
// (the approach, the panel, the greeting): a stop on the way (the stash between the repair vendor and the shop) is not
// free, so one stash stop beats two (U: the value is a judgment, nothing measures it; it only breaks ties between equal
// walks). Walk is the distance between two stops; nil = straight line, the navigator's path length when it can price one.
type routeCosts struct {
	StopPenalty int
	Walk        func(a, b input.Point) int
}

// defaultRouteCosts is the straight-line price list with the stop penalty of the judgment above.
var defaultRouteCosts = routeCosts{StopPenalty: 10}

func (c routeCosts) walk(a, b input.Point) int {
	if c.Walk != nil {
		return c.Walk(a, b)
	}
	return int(math.Round(math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))))
}

// PathCoster is what a navigator offers when it can price a walk on the map it knows: the path length in subtiles, false
// when it has no map or no path (the route then falls back to the straight line for the whole visit).
type PathCoster interface {
	PathCost(from, to input.Point) (int, bool)
}

// routeResult is the chosen order, the visited stops in order (consecutive duplicates merged, "" chores walk nowhere)
// and the walk cost in subtiles.
type routeResult struct {
	Order []string
	Stops []string
	Cost  int
}

// routeOrder picks, among the orders that satisfy the edges, the one with the shortest walk from start over the role
// positions; equal costs keep the earliest original order, so the result is deterministic (invariant T8) and equals the
// input order when every position is the same. Exhaustive over the reachable subsets (DP); more than 22 chores or an
// edge cycle returns the input order unchanged.
func routeOrder(names []string, edges [][2]string, pos map[string]input.Point, start input.Point) routeResult {
	return routeOrderWith(names, edges, pos, start, defaultRouteCosts)
}

// routeOrderWith is routeOrder with an explicit price list.
func routeOrderWith(names []string, edges [][2]string, pos map[string]input.Point, start input.Point, rc routeCosts) routeResult {
	n := len(names)
	keep := func() routeResult {
		r := routeResult{Order: slices.Clone(names)}
		r.Stops, r.Cost = walkOf(names, pos, start, rc)
		return r
	}
	if n == 0 || n > 22 {
		return keep()
	}
	index := map[string]int{}
	for i, nm := range names {
		index[nm] = i
	}
	pre := make([]uint32, n) // chores that must come first
	for _, e := range edges {
		a, aok := index[e[0]]
		b, bok := index[e[1]]
		if aok && bok && a != b {
			pre[b] |= 1 << a
		}
	}
	roles := []string{""}
	for _, nm := range names {
		if r := choreRole[nm]; !slices.Contains(roles, r) {
			roles = append(roles, r)
		}
	}
	roleIdx := func(nm string) int { return slices.Index(roles, choreRole[nm]) }
	type key struct {
		mask uint32
		last int
	}
	type val struct {
		cost int
		next int // chosen next chore, -1 at the end
		ok   bool
	}
	memo := map[key]val{}
	full := uint32(1)<<n - 1
	dist := func(a, b string) int {
		if a == b || a == "" || b == "" {
			return 0
		}
		pa, pb := pos[a], pos[b]
		return rc.StopPenalty + rc.walk(pa, pb)
	}
	startDist := func(role string) int {
		if role == "" {
			return 0
		}
		p := pos[role]
		return rc.StopPenalty + rc.walk(start, p)
	}
	var solve func(mask uint32, last int) val
	solve = func(mask uint32, last int) val {
		if mask == full {
			return val{next: -1, ok: true}
		}
		k := key{mask, last}
		if v, hit := memo[k]; hit {
			return v
		}
		best := val{next: -1}
		for i := range n {
			if mask&(1<<i) != 0 || pre[i]&^mask != 0 {
				continue
			}
			ri := roleIdx(names[i])
			step, nl := 0, last
			if roles[ri] != "" {
				if last == 0 {
					step = startDist(roles[ri]) // the first walk starts at the hero
				} else {
					step = dist(roles[last], roles[ri])
				}
				nl = ri
			}
			rest := solve(mask|1<<i, nl)
			if !rest.ok {
				continue
			}
			if c := step + rest.cost; !best.ok || c < best.cost {
				best = val{cost: c, next: i, ok: true}
			}
		}
		memo[k] = best
		return best
	}
	if !solve(0, 0).ok {
		return keep() // a cycle in the edges: never block the visit, run the list as given
	}
	var res routeResult
	mask, last := uint32(0), 0
	for mask != full {
		v := solve(mask, last)
		res.Order = append(res.Order, names[v.next])
		if ri := roleIdx(names[v.next]); roles[ri] != "" {
			last = ri
		}
		mask |= 1 << v.next
	}
	res.Stops, res.Cost = walkOf(res.Order, pos, start, rc)
	return res
}

// walkOf is the stop sequence and walk cost of running the chores in the given order.
func walkOf(order []string, pos map[string]input.Point, start input.Point, rc routeCosts) (stops []string, cost int) {
	cur, at := "", start
	for _, nm := range order {
		r := choreRole[nm]
		if r == "" || r == cur {
			continue
		}
		p := pos[r]
		cost += rc.StopPenalty + rc.walk(at, p)
		cur, at = r, p
		stops = append(stops, r)
	}
	return stops, cost
}

// routePositions reads the stop positions of the act from the presets (absolute subtiles of this game). ok is false when a
// role the visit needs has no preset, and the fixed list runs.
func (t *Town) routePositions(a *Act) map[string]input.Point {
	pos := map[string]input.Point{
		"repair": t.npcPreset(a.Repair...),
		"shop":   t.npcPreset(a.Shop...),
		"cain":   t.npcPreset(a.Cain...),
		"stash":  t.objPreset(ObjStash),
	}
	return pos
}

// reorderChores applies stage 5 to the visit's chore list: it returns the names in the order to run and logs the route.
func (t *Town) reorderChores(s *d2.Snapshot, a *Act, names []string, settleFirst bool) []string {
	pos := t.routePositions(a)
	routed, unplaced := splitUnplaced(names, pos)
	for _, nm := range unplaced {
		t.e.Log.Info(fmt.Sprintf("town plan: route leaves out chore %s (no preset position for %s)", nm, choreRole[nm]))
		delete(pos, choreRole[nm])
	}
	// Nothing left to price (no stop of this visit has a preset): the walk gives no reason to leave the fixed list, and
	// its settle-before-shop order is the one whose second run is idle (TestChoresAct1AtRouteStage).
	if !hasStop(routed) {
		t.e.Log.Info("town plan: route kept the list (no stop of this visit has a preset position)")
		return names
	}
	start := input.Point{X: s.Player.X, Y: s.Player.Y}
	rc := t.routePrices(pos, start)
	edges := routeEdges(names, settleFirst)
	res := routeOrderWith(routed, edges, pos, start, rc)
	res.Order = reinsertChores(res.Order, unplaced, edges)
	t.lastStops.Store(&res.Stops)
	t.e.Log.Info(fmt.Sprintf("town plan: route stops=%s cost=%d settle-first=%v order=%s", strings.Join(res.Stops, ","), res.Cost, settleFirst, strings.Join(res.Order, ",")))
	return res.Order
}

// splitUnplaced separates the chores whose stop has no preset (Deckard Cain before his rescue, a preset the map does not
// give) from those the route can price: one missing NPC does not cost the whole route.
func splitUnplaced(names []string, pos map[string]input.Point) (routed, unplaced []string) {
	for _, nm := range names {
		if r := choreRole[nm]; r != "" && pos[r] == (input.Point{}) {
			unplaced = append(unplaced, nm)
			continue
		}
		routed = append(routed, nm)
	}
	return routed, unplaced
}

// hasStop reports whether any of the chores walks to a stop.
func hasStop(names []string) bool {
	return slices.ContainsFunc(names, func(nm string) bool { return choreRole[nm] != "" })
}

// reinsertChores puts the chores that were left out of the route back into order, each right after the last chore it must
// follow (at the front when it follows none), in their original relative order.
func reinsertChores(order, left []string, edges [][2]string) []string {
	order = slices.Clone(order)
	for _, m := range left {
		at := 0
		for i, c := range order {
			for _, e := range edges {
				if e[0] == c && e[1] == m {
					at = i + 1
				}
			}
		}
		order = slices.Insert(order, at, m)
	}
	return order
}

// routePrices is the price list of this visit: the navigator's path length when every leg between the start and the role
// positions can be priced (all or nothing, so the legs stay comparable), else the straight line.
func (t *Town) routePrices(pos map[string]input.Point, start input.Point) routeCosts {
	rc := defaultRouteCosts
	pc, ok := t.mv.(PathCoster)
	if !ok {
		return rc
	}
	memo := map[[2]input.Point]int{}
	pts := []input.Point{start}
	for _, p := range pos {
		pts = append(pts, p)
	}
	for _, a := range pts {
		for _, b := range pts {
			if a == b {
				continue
			}
			c, ok := pc.PathCost(a, b)
			if !ok {
				return rc
			}
			memo[[2]input.Point{a, b}] = c
		}
	}
	rc.Walk = func(a, b input.Point) int { return memo[[2]input.Point{a, b}] }
	return rc
}
