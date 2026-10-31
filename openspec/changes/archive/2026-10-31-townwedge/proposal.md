## Why

Both overnight d2d soaks (seed 5, docs/probes/navsoak-compare.md) ended in the same permanent wedge in the Rogue Encampment: after one failed roam out of a town spot ("roam leg failed ... {138 66} is unreachable from {145 68}") nav marked the town to Blood Moor link bad ("nav edge marked bad 1 2"), every later route from the town failed ("no route from level 1"), and 950 save-and-exit re-entries did not clear the mark. navsoak lost 60 wall minutes (80 virtual h), navsoak2 167 wall minutes (263 virtual h).

## What Changes

- internal/nav/route.go, exit.go: bad edges and bad gates expire (knob `nav.badEdgeTTLSec`, default 300 s of game time) and are dropped when the game is re-entered; an edge the level map itself shows is never marked bad by a failed walk.
- internal/nav/exit.go: on a fully known map (d2d), a gate that is on the map but unreachable from the character's cell is a pocket, not "no exit": walk to the nearest connected cell instead of exploring and failing with ErrNoExit.
- Offline tests in internal/nav/townwedge_test.go; docs/townwedge.md.
- Delta: bot-loop ADDED `A failed walk does not condemn a route for good`.
