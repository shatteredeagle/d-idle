# Town resupply in Gate 2 (townshop)

> status: current. Tags: V = counted in the Gate 2 bot log (read-only), S = code, U = unverified.

## Root cause of NOKILL 15 (V)
Log gate2-20261102-0823, wall 09:43:45 to 09:44:30, about 29x speed.
- The 27 `shop: bought` lines are 8 to 15 ms of wall each (V). Single clicks are not the cost; no quantity verb is used or needed (nothing in docs/d2d-wire.md or the town code offers one: U).
- The cost is a trip loop. In the field the dump lost stat 70 of the tome (`inv-tp-dump ... counted=0 raw70=no70`, the town read the same tome as 13 to 19, V). After the 2 minute debounce the verdict flipped (`tp-debounce-flip`), ToTown logged `no portal source (empty)` and walked via the neighbour field and the waypoint, trips at 09:43:45, 09:43:52, 09:44:05, 09:44:29 (reasons mp belt, no town portal scroll).
- `NotePortalEmpty` makes the shop ignore the stat read, so it bought 5 scrolls and then 2 more on a tome that read 13 (700 gold of 914). The poverty raised the next potion trips. After the second empty visit the watchdog (fixprogress) went quiet.
- The `nav stuck` lines in town (108,58) are sidestep/spiral recoveries within 10 to 100 ms; not a stall (V).

## Fix (S, offline tests only)
- `ChoreBudgetS` is now also checked inside the potion and scroll buy loops (it was only checked between chores); on expiry one line `town: resupply budget hit`, what was bought stays, kit order (potions, scrolls, rest) unchanged.
- After a portal-empty report with a readable tome (stat 70 > 0) the visit buys `emptyRestock` (2) scrolls and none at 20: a proof the source works. Gate 1's stale-stat case still gets a restock.

## Not fixed / U
- Why the field read drops stat 70 at counts of 13 to 19 (reader or d2d side); the debounce span (120 s virtual) is shorter than these drop-outs at 29x speed.
- Live checklist: a slot with a tome of 13 scrolls, force `NotePortalEmpty`, one Chores call: expect 2 `shop: bought` scrolls and no `budget hit`; with `choreBudgetS` 30, a large shopping list stops with `town: resupply budget hit`.
