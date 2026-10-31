package nav

import "d2lod-bot/internal/input"

// act2Hints: outdoor Act 2 links that levels.txt Vis does not list (they are level-edge crossings, not warps). Found
// by ACT2LIVE (live on i1: `navlive route 40 41` said "no route" because Vis of 40 holds only 47/50). Tag: S (Arreat map
// order); each hop is confirmed or marked bad by the first ExitTo. Canyon of the Magi (46): levels.txt Vis (V, dumped
// from the 1.14d tables) lists ONLY the seven tombs 66..72, so it has no data-backed outdoor neighbour. ACT2LIVE2
// found the desert chain 43-44-45 by Adjacent contacts (V) and the Valley of Snakes (45) a dead end for plain nav
// (its only listed exit is Dest 58; a teleport west was blocked by rocks). The 45-46 hop below is therefore U: the
// first ExitTo decides (a failed hop is marked bad and routed around). The reliable way to 46 is its waypoint
// (TakeWaypoint(46)) once any visit activated it; the tombs are in levels.txt Vis.
var act2Hints = [][2]int{
	{40, 41}, // Lut Gholein - Rocky Waste
	{41, 42}, // Rocky Waste - Dry Hills
	{42, 43}, // Dry Hills - Far Oasis
	{43, 44}, // Far Oasis - Lost City
	{44, 45}, // Lost City - Valley of Snakes
	{45, 46}, // Valley of Snakes - Canyon of the Magi (U, see above)
}

// knownWaypointPos: waypoint positions for levels where the preset streams in late. Lut Gholein (40): object 156 at
// (5069,5084), V live 2026-10-29 (docs/act2.md nav2). Town layouts can shift with the game seed (see the town seed
// tests), so this is a first guess to walk toward, never a click target (U for other seeds).
var knownWaypointPos = map[int]input.Point{
	40: {X: 5069, Y: 5084},
}
