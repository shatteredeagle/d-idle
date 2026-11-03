# townshop: shorter town resupply (Gate 2 NOKILL 15)

## Why
Gate 2 (docs/research/gate2-analysis-20261102.md, fix 3) failed NOKILL with 15 virtual minutes in town chores. Reading the run log (docs/townshop.md): the 27 `shop: bought` lines are about 10 ms of wall each, so single-unit clicks are not the cost. The minutes were a trip loop: the field read lost the tome's stat 70, ToTown found "no portal source", walked and took the waypoint, the shop (told "empty") bought 5 and then 2 more scrolls on top of a tome that read 13, which drained the gold and raised more potion trips.

## What Changes
- A chore time budget inside the buy loops (`ChoreBudgetS`), one line `town: resupply budget hit`.
- When the game said empty but the tome still reads scrolls, the visit buys a 2-scroll proof, not a refill.

## Overlap (declared)
Overlap (declared): `internal/town/chores.go` and `town.go` belong to fixprogress and `docs/botlogs.md` to an earlier card; this card edits them (a deadline field, one helper, one log row) and does not list them in owns. fixprogress tests stay green.

## Not done
Live check (U); the root cause of the dropped stat 70 in the field (d2d reader) is not fixed here.
