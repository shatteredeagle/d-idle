# navsoak vs navsoak2: paired soak comparison (2026-10-31)

> status: analysis of two hidden d2d soaks, read-only. Handoffs: branches agent/navsoak, agent/navsoak2 (docs/probes/*/README.md). Tags: VERIFIED = read from the logs on the worker host; INFERRED = reading of them. n=1 paired seed: not a statistical claim.

## Runs
- navsoak: build 279a163 (nav-stall fix, BASICSFIX, champion caution), Sorceress, seed 5, 240 wall min, ended clean (exit 0, 0 supervisor restarts). navsoak2: build d82bd47 (plus KITGATE, M1LOOT), same seed and flags. VERIFIED.
- navsoak2 was still running when analysed (06:30, wall min 212 of 240); its wedge began at wall min 45, so the rest only adds waiting lines. Final-hour numbers for it are a snapshot. VERIFIED.

## Headline per soak
| | navsoak | navsoak2 |
|---|---|---|
| final clvl / xp / kills | 13 / 125526 / 7216 | 9 / 50219 / 1784 (snapshot) |
| virtual h played (all / before wedge) | 117.9 / 38.0 | 273 (snapshot) / 9.7 |
| progress ends (wall min, state) | 179, wedged in town | 45, wedged in town |
| kills per virtual h before wedge | 190 | 185 |
| deaths | 1 (clvl 4, Cave Level 1, wall min 3.4) | 0 |
| quests | Den progress, Blood Raven (sisters_burial) done at wall 68, search_for_cain started | same, Blood Raven done at wall 23 |
VERIFIED. Quest names are the log's: Dark Wood/Inifuss (13 and 7 area entries of Dark Wood and Black Marsh in navsoak, none in navsoak2) and the Forgotten Tower were never done; route guard kept the tower blocked (no merc below clvl 20, or fewer than 2 mana or 4 healing potions).

## The combat livelock is gone
VERIFIED: "combat no progress, holding" 0 in both. "stepping away" 791 (navsoak) and 139 (navsoak2), "combat move failed" 6513 / 59 of which "nav: no path" 6455 / 43. The virtual clock never froze: 0 pace intervals without advance (240 and 211 pace lines). No supervisor restart in either. Answer: yes, gone as a hang. Residue: in navsoak the nav-no-path count is large (about 170 per virtual h against 6 in navsoak2); it did not stop progress (INFERRED: it comes from Burial Grounds and Mausoleum visits, navsoak entered Burial Grounds 338 times, navsoak2 31).

## Where both stall: a permanent nav wedge, not a timer wait
VERIFIED, same signature in both (navsoak 05:28:40, navsoak2 03:43:55):
1. "back to field: portal in town" then "portal did not lead to the level" (this had happened 69 and 29 times earlier and recovered by walking).
2. "nav no exit known yet, exploring" from level 1, "roam: coverage points level 1, exits 0", "roam leg failed ... {138 66} is unreachable from {145 68}" (identical coordinates), "roam: done visited 293 of 397" (navsoak).
3. "nav edge marked bad from 1 to 2", then "no route from level 1 to N" for every area.
After that: 0 "Entered" lines, 0 roam, 0 further exploration, 0 purchases; the bot sits in the Rogue Encampment (area 1) printing "area gate refused the walk / area skipped (unreachable) / leveling target not reachable / Waiting: X banned for 3 more min / area wait: all areas banned/skipped, farming X anyway". navsoak: 51.9k "no eligible reachable leveling area" lines after the wedge and 29 before it; navsoak2 173k and 7. navsoak lost 60 wall min and 80 virtual h; navsoak2 167 wall min and 263 virtual h. It also survived 950 save-and-exit re-entries in navsoak, so the bad edge is not cleared by re-entering the game. The earlier "no eligible area" lines (first at wall 22 and 25) were transient timer waits (dozens of lines, play continued).
INFERRED: the character stood in an unreachable pocket of the town map at (145,68) after a failed portal, so the roam found no exit and the nav graph condemned the town-to-Blood-Moor edge for good. The "message" the user saw near 05:43 is this wedge, which in navsoak started at 05:28.

## A/B
- First divergence VERIFIED: about 15 wall s in, clvl 1, gold 4: navsoak2 logs "recovery: kit gate on (no potion or no portal): Blood Moor and Cold Plains only" where navsoak goes on with its emergency portal and the full area list.
- First death: navsoak clvl 4, Cave Level 1, "Recovery: farming gold", belt 0 hp, 0 portal scrolls, killer Skeleton (not a champion). The clvl-7 Burial Grounds death did not occur in either; navsoak2 had none in 1784 kills (kit gate on 20 times, off 20 times). One paired run: n=1.
- Recovery area entries per virtual h before the wedge (INFERRED from minute-line step): navsoak 27 (Cold Plains 309, Cave Level 1 193, Blood Moor 157, Stony Field 102, Burial Grounds 43 in total), navsoak2 8.6 (Cold Plains 24, Blood Moor 18, Cave Level 1 12). Recovery share of virtual minutes 43% vs 28%; town chores 32% vs 48%.
- Purchases before the wedge, per virtual h: healing potions 4.6 vs 7.8, TP scrolls 3.5 vs 5.3, mana potions 4.3 vs 3.7; first TP bought at wall 2.2 (clvl 4) vs 2.1 (clvl 3). Town trips per virtual h 9.2 vs 5.2 (navsoak 666 trips over the whole run, 348 before the wedge).
- Gold: median 96 vs 113, below 100 in 53% vs 44% of minutes, max 691 vs 806: both stay under the 500 kit line almost always. Belt potions at 0 in 33% vs 25% of minutes; "no healing potion in the belt" 59 vs 27 per virtual h.
- Speed to clvl 9: navsoak at wall 31 (1549 kills, about 8.4 virtual h), navsoak2 at wall 41 (1556 kills, about 8.6 virtual h); equal in virtual time, wall differs with host load. XP per kill equal at equal kills (INFERRED).
- Improved (n=1): no early death, fewer area changes and trips, fewer empty-belt minutes. Worse or unchanged: gold starvation unchanged (below 100 gold half the time), more time in town chores. Cannot conclude: death rates, XP/h at higher levels (navsoak2 never passed clvl 9), whether KITGATE helps beyond one seed.

## Backlog findings
1. Permanent town-exit wedge (card `townwedge`): clear a bad town edge on a timer or game re-entry, re-seed exits from the d2dworld "3 adjacent entries" the log already prints, and treat "exits 0" in a town as an error (log above, 2 of 2 runs). Needs a unit test with a fake graph and a live check.
2. Wall-state guard (card `nowall`): when "no eligible reachable leveling area" repeats for more than a few virtual minutes with 0 "Entered", escalate (restart the game, then supervisor restart) instead of 170k identical lines; the supervisor never fired because the bot kept running.
3. Gold starvation (card `goldfloor`): median gold about 100, kit 500 never met, 33% of minutes with an empty belt and 2239 "no healing potion" lines in navsoak. Candidates: sell loot earlier, lower the kit target at low clvl, stop recovery trips that cost more than they earn.

## Next
Run the paired tooling (docs/ab-design.md, tools/ab/paired.py) with at least 8 seeds per arm, 2 to 4 virtual h each, after finding 1 is fixed (else the wedge ends every run early, as here). Measure first death, recovery area churn, trips per h and empty-belt share. A fresh navsoak-style soak of 240 min on the fixed build comes after that.
