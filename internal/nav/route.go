package nav

import (
	"fmt"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
)

// Edge sources for Route, cheapest first.
const (
	srcSeen  = 1 // seen in this game: a tile exit or border gap in the level map, or a crossing the character made
	srcVis   = 2 // levels.txt Vis columns: neighbour hints from the game data (not a complete or directed list)
	srcHint  = 3 // static Act 1 table below
	maxRoute = 40
)

// act1Hints is a FALLBACK adjacency list of Act 1 (undirected), used only for levels the character has not
// seen yet. It comes from the Diablo II level layout (wiki / Kolbot area list) and is UNVERIFIED in this client
// except for 1-2 and 2-8, 2-3 (see docs/nav.md). Runtime knowledge always wins: a hint that fails an ExitTo is
// marked bad and routed around.
var act1Hints = [][2]int{
	{1, 2},   // Rogue Encampment - Blood Moor (VERIFIED LIVE)
	{2, 8},   // Blood Moor - Den of Evil
	{2, 3},   // Blood Moor - Cold Plains
	{3, 9},   // Cold Plains - Cave level 1
	{9, 13},  // Cave 1 - Cave 2
	{3, 17},  // Cold Plains - Burial Grounds
	{17, 18}, // Burial Grounds - Crypt
	{17, 19}, // Burial Grounds - Mausoleum
	{3, 4},   // Cold Plains - Stony Field
	{4, 10},  // Stony Field - Underground Passage 1
	{10, 5},  // Underground Passage 1 - Dark Wood (the only way on foot: levels.txt Cave 3 Vis = 4, 5)
	{4, 38},  // Stony Field - Tristram
	{10, 14}, // Underground Passage 1 - Underground Passage 2 (levels.txt Cave 3 Treasure)
	{5, 6},   // Dark Wood - Black Marsh
	{6, 11},  // Black Marsh - Hole 1
	{11, 15}, // Hole 1 - Hole 2
	{6, 7},   // Black Marsh - Tamoe Highland
	{6, 20},  // Black Marsh - Forgotten Tower (levels.txt Tower 2 Vis = 6, 21; NOT Tamoe Highland)
	{7, 12},  // Tamoe Highland - Pit 1
	{12, 16}, // Pit 1 - Pit 2
	{20, 21}, // Forgotten Tower - Tower Cellar 1 (to 25)
	{21, 22}, {22, 23}, {23, 24}, {24, 25},
	{7, 26},  // Tamoe Highland - Monastery Gate
	{26, 27}, // Monastery Gate - Outer Cloister
	{27, 28}, // Outer Cloister - Barracks
	{28, 29}, // Barracks - Jail 1
	{29, 30}, {30, 31},
	{31, 32}, // Jail 3 - Inner Cloister
	{32, 33}, // Inner Cloister - Cathedral
	{33, 34}, // Cathedral - Catacombs 1
	{34, 35}, {35, 36}, {36, 37},
}

type edgeKey [2]int

// learn records a crossing the character made: both directions work.
func (n *Navigator) learn(from, to int) {
	if from == 0 || to == 0 || from == to {
		return
	}
	if n.seen == nil {
		n.seen = map[edgeKey]bool{}
	}
	n.seen[edgeKey{from, to}] = true
	n.seen[edgeKey{to, from}] = true
	kept := n.badGates[:0]
	for _, b := range n.badGates {
		if !(b.from == from && b.to == to) {
			kept = append(kept, b)
		}
	}
	n.badGates = kept
	delete(n.bad, edgeKey{from, to})
	delete(n.bad, edgeKey{to, from})
}

// badEdge is a guessed edge a walk failed on. Unless the knob BadEdgeTTLSec is 0 it expires: after the TTL (game
// time), or when the game was re-entered (the level map of from is another object then: d2dworld forgets its levels).
// TOWNWEDGE (soak logs navsoak, navsoak2): one failed roam out of a cut-off town spot marked 1->2 bad for good, every
// later route from the town failed, and 950 save-and-exit re-entries did not clear it.
type badEdge struct {
	until time.Time // zero: for the rest of the run
	lm    *d2map.LevelMap
}

func (n *Navigator) badTTL() time.Duration {
	if n.tune == nil {
		return time.Duration(env.DefaultNavTuning().BadEdgeTTLSec * float64(time.Second))
	}
	return time.Duration(n.tune().BadEdgeTTLSec * float64(time.Second))
}

func (n *Navigator) markBad(from, to int) {
	if n.bad == nil {
		n.bad = map[edgeKey]badEdge{}
	}
	b := badEdge{}
	if ttl := n.badTTL(); ttl > 0 {
		b = badEdge{until: n.clk.Now().Add(ttl), lm: n.lvl(from)}
	}
	n.bad[edgeKey{from, to}] = b
	n.log.Info("nav edge marked bad", "from", from, "to", to, "ttlSec", int(n.badTTL().Seconds()))
}

// isBad reports a live bad mark. It never writes the map (Route may run beside the walking goroutine).
func (n *Navigator) isBad(from, to int) bool {
	b, ok := n.bad[edgeKey{from, to}]
	if !ok {
		return false
	}
	if b.until.IsZero() {
		return true
	}
	if !n.clk.Now().Before(b.until) {
		return false
	}
	return b.lm == nil || n.lvl(from) == b.lm
}

// edgeShown reports that the level map of from itself shows a way to level to (tile exit, or border cells of a
// neighbour d2dworld linked): such an edge exists whatever one walk did, so a failed walk is not a reason to mark it.
func (n *Navigator) edgeShown(from, to int) bool {
	if lm := n.lvl(from); lm != nil && lm.Grid != nil && hasGate(lm, to) {
		return true
	}
	return n.seen[edgeKey{from, to}]
}

func (n *Navigator) levelNow() int {
	if n.st == nil {
		return 0
	}
	s, err := n.st.Snapshot()
	if err != nil || s.State != d2.StateInGame || s.Player == nil {
		return 0
	}
	return s.Player.Area
}

// edges collects every known directed edge with the strength of its source.
func (n *Navigator) edges() map[int]map[int]int {
	adj := map[int]map[int]int{}
	add := func(a, b, src int) {
		if a == b || b == 0 || n.isBad(a, b) {
			return
		}
		if adj[a] == nil {
			adj[a] = map[int]int{}
		}
		if old, ok := adj[a][b]; !ok || src < old {
			adj[a][b] = src
		}
	}
	for _, h := range allHints() {
		add(h[0], h[1], srcHint)
		add(h[1], h[0], srcHint)
	}
	if n.e != nil && n.e.Data != nil {
		for _, l := range n.e.Data.Levels {
			for _, v := range l.Vis {
				add(l.ID, v, srcVis)
				add(v, l.ID, srcVis)
			}
		}
	}
	if lv, ok := n.w.(interface{ LevelNos() []int }); ok {
		for _, no := range lv.LevelNos() {
			lm := n.w.Level(no)
			if lm == nil {
				continue
			}
			for _, x := range lm.Exits {
				add(no, x.Dest, srcSeen)
			}
			for _, a := range lm.Adjacent {
				if lm.Grid != nil && len(lm.BorderCells(a.LevelNo)) > 0 {
					add(no, a.LevelNo, srcSeen)
				}
			}
		}
	}
	for k := range n.seen {
		add(k[0], k[1], srcSeen)
	}
	return adj
}

// avoidCost is the extra cost of passing through a level SetAvoidLevels named: a detour of up to this many seen edges
// is taken to stay out, but a destination behind it stays reachable (a cost, not a wall).
const avoidCost = 30

// SetAvoidLevels names levels (dungeon mouths) a walk should not pass through: Route prices entering them at avoidCost
// and NearMouth answers for tile exits while the set is non-empty. nil or empty clears it. KITGATE: a recovering
// character without the kit keeps out of the Den, caves and Burial Grounds.
func (n *Navigator) SetAvoidLevels(m map[int]bool) {
	if len(m) == 0 {
		n.avoid.Store(nil)
		return
	}
	c := make(map[int]bool, len(m))
	for k, v := range m {
		c[k] = v
	}
	n.avoid.Store(&c)
}

func (n *Navigator) avoiding() bool { p := n.avoid.Load(); return p != nil && len(*p) > 0 }

func (n *Navigator) avoided(level int) bool {
	p := n.avoid.Load()
	return p != nil && (*p)[level]
}

// Route lists the levels to enter, in order, to get from one level to another over the exits this game has
// shown (tile exits, border gaps, crossings already made), then levels.txt neighbours, then the static Act 1
// fallback. Cheapest total wins (a seen edge costs 1, a hint up to 3). Acts are not connected (quest doors and
// the act transitions are not walkable edges): the error is ErrNoExit.
func (n *Navigator) Route(from, to int) ([]int, error) {
	if from == to {
		return nil, nil
	}
	adj := n.edges()
	dist := map[int]int{from: 0}
	prev := map[int]int{}
	done := map[int]bool{}
	for len(done) < maxRoute*10 {
		cur, best := -1, 1<<30
		var keys []int
		for k := range dist {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		for _, k := range keys {
			if !done[k] && dist[k] < best {
				cur, best = k, dist[k]
			}
		}
		if cur < 0 || cur == to {
			break
		}
		done[cur] = true
		for nx, c := range adj[cur] {
			if nx != to && n.avoided(nx) {
				c += avoidCost
			}
			if d, ok := dist[nx]; !ok || best+c < d {
				dist[nx], prev[nx] = best+c, cur
			}
		}
	}
	if _, ok := dist[to]; !ok {
		return nil, fmt.Errorf("%w: no route from level %d to %d", ErrNoExit, from, to)
	}
	var rev []int
	for c := to; c != from; c = prev[c] {
		rev = append(rev, c)
	}
	for l, r := 0, len(rev)-1; l < r; l, r = l+1, r-1 {
		rev[l], rev[r] = rev[r], rev[l]
	}
	return rev, nil
}

// lvl is the level map of this game for a level number, nil when there is no world source or the level is unseen.
func (n *Navigator) lvl(no int) *d2map.LevelMap {
	if n.w == nil {
		return nil
	}
	return n.w.Level(no)
}
