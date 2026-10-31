## Why

Walk is 13.8 of the 18 seconds per kill on d2d (77 percent, docs/d2d-soak-full-fix.md) against about 5 on the original: the longest gaps are border ping-pong between Blood Moor, Cold Plains and Cave 1 (`nav exit`, `nav stuck`) and roaming a swept area (`clear killed nothing` 0.30 per kill against 0.09 on the original, docs/research/per-kill-time-split.md). The empty-area exit exists (`bot.emptyAreaSec`) but defaults to off (docs/killfix.md). This is the largest remaining component of the fidelity gap after the economy fixes.

## What Changes

- internal/nav/roam.go, exit.go: no return to an area left in the last N minutes unless the plan asks for it; roam toward unexplored, spawn-dense tiles instead of the border.
- internal/bot/goal.go: `bot.emptyAreaSec` default on (value from the KILLFIX measurement, docs/killfix.md) for recovery and leveling sweeps.
- A before/after on one binary, 4 seeds x 2 virtual hours per arm, interleaved (tools/xpgap/ab.sh pattern), walk share and `clear killed nothing` per kill reported.
- Delta: early-game-policy ADDED `Leave an empty area`.

## Dependencies

After m1-trips is archived (it owns internal/bot/goal.go) and after xpgap releases soak-slot:3.
