package nav

// Outdoor links of Acts 3 to 5 that levels.txt Vis does not list (level-edge crossings, like act2Hints). Tags: the
// order of the chains is S (waypoint order of the tables, docs/act3.md/act4.md/act5.md, plus the Arreat Summit maps);
// each edge is U until the first ExitTo, which confirms it or marks it bad (nav then routes around). Links that Vis
// already lists (Durance 83-100-101-102, Kurast Sewers, River of Flame 107-108, the Frozen River/Drifter Cavern
// spurs) are deliberately not repeated here. No edge was walked live.
var act3Hints = [][2]int{
	{75, 76}, // Kurast Docktown - Spider Forest
	{76, 77}, // Spider Forest - Great Marsh
	{77, 78}, // Great Marsh - Flayer Jungle
	{78, 79}, // Flayer Jungle - Lower Kurast
	{79, 80}, // Lower Kurast - Kurast Bazaar
	{80, 81}, // Kurast Bazaar - Upper Kurast
	{81, 82}, // Upper Kurast - Kurast Causeway
	{82, 83}, // Kurast Causeway - Travincal
}

// act4Hints: Pandemonium Fortress to the City of the Damned; River of Flame (107) and the Chaos Sanctuary (108) are
// Vis links (V, docs/act4.md).
var act4Hints = [][2]int{
	{103, 104}, // Pandemonium Fortress - Outer Steppes
	{104, 105}, // Outer Steppes - Plains of Despair
	{105, 106}, // Plains of Despair - City of the Damned
}

// act5Hints: Harrogath to the Ancients' Way. 113 -> 115 and 115 -> 117 skip the Frozen River and Drifter Cavern spurs.
var act5Hints = [][2]int{
	{109, 110}, // Harrogath - Bloody Foothills
	{110, 111}, // Bloody Foothills - Frigid Highlands
	{111, 112}, // Frigid Highlands - Arreat Plateau
	{112, 113}, // Arreat Plateau - Crystalline Passage
	{113, 115}, // Crystalline Passage - Glacial Trail
	{115, 117}, // Glacial Trail - Frozen Tundra
	{117, 118}, // Frozen Tundra - Ancients' Way
}

// allHints is every hand-kept outdoor link, Acts 1 to 5.
func allHints() [][2]int {
	var all [][2]int
	for _, h := range [][][2]int{act1Hints, act2Hints, act3Hints, act4Hints, act5Hints} {
		all = append(all, h...)
	}
	return all
}

// Act 3-5 waypoint panel (X4, docs/act3.md): wpPanel in waypoint.go already carries the tab x of acts 3 (230, V live),
// 4 (287) and 5 (344) (the last two U, extrapolated at the act 2 pitch) and IconFrom = 7, which covers Act 3 rows 7
// (Travincal) and 8 (Durance 2) and Act 5 rows 7 (Ancients' Way) and 8 (Worldstone Keep 2); whether those rows need the
// icon click as in Act 2 is U. knownWaypointPos (act2hints.go) is NOT extended for the towns 75, 103 and 109: the
// positions are per-seed and no live reading exists, so a guess would only mislead the walk.
