# hourfix2: the Den-mouth shuttle of the second fresh hour (2026-11-12 17:35)

Tags: V log (h2.log of the second hour, build 0.0.261112171114), S doc, U unverified live.

## What happened (V)
- 17:31:19 `Quest done: den_of_evil`; 17:34:43 the hero left town and walked to the Den mouth (`nav.trace click ... wp 34 of 36`, leg ends `level changed during the walk: 2 -> 8`) although the reward was taken. Cause (V by test): `target0` returned the cached `loop.stepArea` (Den, same clvl) before the claimed-Den exclusion of hourfix change 3 was reached. Fix: drop a cached Den pick once `questClaimed`; test `TestCachedDenPickDropsAfterTheClaim` (fails without the line).
- In the Den the journey flipped to Cold Plains, so the hero walked back out (`nav exit to 2 gate 140,145`). From 17:35:12 to 17:36:43: 47 `arrival bounce`, 92 area hops of 1.2 to 1.7 s each (about 2 min), then a fresh game (Save and Exit, 75 s in town). Every Blood Moor arrival from the Den (136,140) was back in the Den 0.17 to 0.25 s after the priming click (`nav.trace kick`) and the following `cmd move`, in all kick directions (E, S, NE, N) and move lengths 2 to 20 subtiles; the only exits that passed were the town portal. Arrivals into the Den from Blood Moor (unit subtile free) passed (17:22:54, 02:31 of hour 1).
- So d2d re-enters the cave on the first order after a cave EXIT arrival, whatever the move, which is what docs/probes/fid/fid-warp-arrival.md predicts for arrivals whose unit subtile is blocked (hero displaced by `nearest_free`, ring 1, 25 of 52 arrivals per seed differ from game.exe, mostly cave exits). Two inputs can fire it: the walk (needs a click cell in the warp's box) and the priming click. The walk clicks here were up to 20 subtiles away from the unit in every direction, so see bounce3 for the cause.

## Change
Stale cached Den pick dropped after the claim (above). The kick distance and breaker edits of the first version were dropped: agent/bounce3 found the real bounce cause (priming click within 2 cells of a warp) and replaces them.

## Cost
A bounce costs 1.2 to 1.7 s each; the burst 2 min plus the fresh game 75 s in town and a Blood Moor re-clear of 10 min at 160 xp/min (17:37 to 17:47), so about 14 min of the hour. The other town shuttles of that hour (Cold Plains and town every 4 to 5 min, 17:52 to 18:08) are out of scope here.
