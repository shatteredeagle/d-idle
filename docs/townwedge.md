# townwedge: the permanent town-exit wedge

> status: fix merged with card townwedge. Tags: VERIFIED = log line or test, INFERRED = reading. Evidence: docs/probes/navsoak-compare.md (branch agent/soakcompare), logs of navsoak and navsoak2 (seed 5).

## Root cause
- VERIFIED (navsoak2 log, 03:43:55 to :56, one second of wall time): town, a TP trip "portal did not lead to the level" (it had done so 29 times before and always recovered by walking), then ExitTo(3) and ExitTo(2) from the Rogue Encampment, "roam: coverage points level 1 ... exits 0", "nav stuck" at {169 53} and {145 63}, "roam leg failed ... {138 66} is unreachable from {145 68}", every plan from {145 68} ("{179 49}", "{173 103}") unreachable, "roam: done visited 26 of 397", then "nav edge marked bad 1 2".
- VERIFIED (code): ExitTo found no reachable gate, called explore (roam, which cannot find anything on a fully known d2dworld map), returned ErrNoExit, and ToArea marked the edge for good (`markBad`, no expiry, never reset). Route then has no 1 to 2 edge, so "no route from level 1 to N" for every area. 950 save-and-exit re-entries did not clear it because the mark lived in the Navigator, not in the game.
- VERIFIED: "exits 0" is only the count of TILE exits (caves, stairs); the Rogue Encampment leaves by border (d2dworld "3 adjacent entries"), so 0 is normal and is not the fault.
- INFERRED: why {145 68} was cut off. Both runs hit the same cell, the plan failed with the hard-block fallback too (no "blacklisted cell" line in the minute before), so the start cell sat in a closed island of the known grid (a rung 1 side step pushed the character onto a cell the grid shows walled). NOT reproduced: the town fixtures differ per seed. The fix does not depend on this.

## Fix (knob `nav.badEdgeTTLSec`, default 300, 0 = old behaviour)
- internal/nav/route.go: bad edges expire (game time) and are dropped when the level map of the source level is replaced (d2dworld resets its levels on every re-entry); `edgeShown`: an edge the map itself shows (tile exit, linked border, crossed before) is never marked by a failed walk.
- internal/nav/exit.go: bad gates expire the same way; on a fully known map, a gate that is on the map but unreachable from the character's cell is a pocket: `pocketTarget` floods from the gate, picks the nearest connected cell off the walls and `leavePocket` clicks toward it in 10-subtile steps; if that fails ToArea returns ErrNoPath (not ErrNoExit), so nothing is marked.
- internal/env/tuning_nav.go, docs/knobs.md (one table line; owned by m3-routeguard, regenerate on merge).

## Tests (internal/nav/townwedge_test.go)
Pocket left and edge unmarked; unleavable pocket gives a non-ErrNoExit error and the route survives; knob 0 reproduces the wedge (Route stays "no route" 24 h later); expiry after 5 min; clearing on re-entry (new level map object).

## Live verify
(pending)
