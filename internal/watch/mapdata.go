package watch

// Static map data of the orders screen (docs/research/orders-ux-design.md, section 3). One table per act; Act 1 is
// filled, Acts 2 to 5 are empty data until their areas are orderable. Layout (X, Y) is stored, not computed, so a
// hand-tuned layout survives. Nothing here is a game read.
//
// Tags: area ids are V (levels.txt ids as in internal/nav/route.go act1Hints and cmd/d2dview/areas.go, the same table
// the bot's guards use). Edges are the bot's own table (internal/nav/route.go) with dungeon levels folded into one node,
// cross-checked against the d2-sme graph of the design doc: the differences are listed in docs/research/orders-ux-design.md
// (section 11). Veteran clvl ranges and the "why farm here" lines are S (d2-sme, not yet signed off) and are ADVICE only.

// MapNode is one area (or a folded group of dungeon levels) of the act map.
type MapNode struct {
	ID       string // stable key
	Name     string // shown name
	Order    string // area name sent in an order and resolved by the bot (goals names; U for the dungeons); "" = not orderable yet
	Kind     string // town, field, dungeon
	X, Y     int
	Waypoint bool
	IDs      []int  // game area ids the node stands for (the folded levels); the first is the entrance
	Lo, Hi   int    // veteran clvl range, advice only (0 = none; Hi 0 = open ended)
	Why      string // one line, "why farm here"
	Sub      string // second line of a node that has no status (the folded Monastery line)
}

// MapEdge joins two nodes; Dungeon draws it dashed.
type MapEdge struct {
	From, To string
	Dungeon  bool
}

// ActMap is the data of one act.
type ActMap struct {
	Act   int
	Town  string // id of the town node: the start of every route
	Nodes []MapNode
	Edges []MapEdge
}

// Maps holds Acts 1 to 5. Acts 2 to 5 have no nodes yet.
var Maps = map[int]ActMap{
	1: {Act: 1, Town: "camp",
		Nodes: []MapNode{
			{ID: "camp", Name: "Rogue Encampment", Kind: "town", X: 62, Y: 744, Waypoint: true, IDs: []int{1}},
			{ID: "bm", Name: "Blood Moor", Order: "Blood Moor", Kind: "field", X: 62, Y: 680, IDs: []int{2}, Lo: 1, Hi: 8, Why: "safe first xp"},
			{ID: "den", Name: "Den of Evil", Order: "Den of Evil", Kind: "dungeon", X: 62, Y: 616, IDs: []int{8}, Lo: 1, Hi: 8, Why: "quest; many monsters in a small cave, no waypoint"},
			{ID: "cp", Name: "Cold Plains", Order: "Cold Plains", Kind: "field", X: 180, Y: 680, Waypoint: true, IDs: []int{3}, Lo: 6, Hi: 12, Why: "open ground, waypoint"},
			{ID: "cave", Name: "Cave 1 to 2", Order: "Cave Level 1", Kind: "dungeon", X: 180, Y: 744, IDs: []int{9, 13}, Lo: 6, Hi: 10, Why: "small, quick"},
			{ID: "bg", Name: "Burial Grounds", Order: "Burial Grounds", Kind: "field", X: 298, Y: 680, IDs: []int{17}, Lo: 8, Hi: 14, Why: "Blood Raven and her archers shoot from range"},
			{ID: "crypt", Name: "Crypt", Order: "Crypt", Kind: "dungeon", X: 298, Y: 616, IDs: []int{18, 19}, Lo: 8, Hi: 14, Why: "small, a chest"},
			{ID: "sf", Name: "Stony Field", Order: "Stony Field", Kind: "field", X: 180, Y: 616, Waypoint: true, IDs: []int{4}, Lo: 10, Hi: 16, Why: "open ground, easy to path"},
			{ID: "up", Name: "Underground Pass", Kind: "dungeon", X: 180, Y: 552, IDs: []int{10, 14}, Sub: "to Dark Wood"},
			{ID: "dw", Name: "Dark Wood", Order: "Dark Wood", Kind: "field", X: 180, Y: 488, Waypoint: true, IDs: []int{5}, Lo: 10, Hi: 16, Why: "the Inifuss tree is here"},
			{ID: "bmarsh", Name: "Black Marsh", Order: "Black Marsh", Kind: "field", X: 180, Y: 424, Waypoint: true, IDs: []int{6}, Lo: 12, Hi: 18, Why: "waypoint, branches to the Hole and the Forgotten Tower"},
			{ID: "hole", Name: "Hole 1 to 2", Order: "Hole Level 1", Kind: "dungeon", X: 62, Y: 424, IDs: []int{11, 15}, Lo: 12, Why: "dense packs"},
			{ID: "ftw", Name: "Forgotten Tower", Order: "Forgotten Tower", Kind: "dungeon", X: 298, Y: 424, IDs: []int{20}, Lo: 15, Hi: 22, Why: "the Countess' runes; a long chain, no waypoint"},
			{ID: "cell", Name: "Tower Cellars 1-5", Kind: "dungeon", X: 298, Y: 360, IDs: []int{21, 22, 23, 24, 25}, Lo: 15, Hi: 22, Why: "the Countess is on level 5", Sub: "not yet"},
			{ID: "tam", Name: "Tamoe Highland", Order: "Tamoe Highland", Kind: "field", X: 180, Y: 360, IDs: []int{7}, Lo: 14, Hi: 20, Why: "leads on to the Monastery"},
			{ID: "pit", Name: "Pit 1 to 2", Order: "Pit Level 1", Kind: "dungeon", X: 62, Y: 360, IDs: []int{12, 16}, Lo: 16, Hi: 24, Why: "dense, no quest"},
			{ID: "mg", Name: "Monastery Gate", Kind: "field", X: 180, Y: 296, IDs: []int{26}},
			{ID: "oc", Name: "Outer Cloister", Kind: "field", X: 180, Y: 232, Waypoint: true, IDs: []int{27}},
			{ID: "line", Name: "Monastery line", Kind: "dungeon", X: 180, Y: 168, IDs: []int{28, 29, 30, 31, 32, 33, 34, 35, 36, 37}, Sub: "to Andariel"},
		},
		Edges: []MapEdge{
			{"camp", "bm", false}, {"bm", "den", true}, {"bm", "cp", false}, {"cp", "cave", true}, {"cp", "bg", false},
			{"bg", "crypt", true}, {"cp", "sf", false}, {"sf", "up", true}, {"up", "dw", true}, {"dw", "bmarsh", false},
			{"bmarsh", "hole", true}, {"bmarsh", "ftw", true}, {"ftw", "cell", true}, {"bmarsh", "tam", false},
			{"tam", "pit", true}, {"tam", "mg", false}, {"mg", "oc", false}, {"oc", "line", true},
		},
	},
	2: {Act: 2}, 3: {Act: 3}, 4: {Act: 4}, 5: {Act: 5},
}

// OrderableAreas is the list of areas a "go to" order may name: the Act 1 nodes that carry an order name. U: the bot
// resolves the name (data catalog, plan table, goals.Catalog); a name that does not resolve ends the order failed with
// the bot's own reason, it never walks anywhere.
func OrderableAreas() []string {
	var out []string
	for _, n := range Maps[1].Nodes {
		if n.Order != "" {
			out = append(out, n.Order)
		}
	}
	return out
}
