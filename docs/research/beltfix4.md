# BELTFIX4: Gate 1 20261103-0053 BELT (7 vmin) and XP/h (8162)

> status: current. Analysis of the third Gate 1 run on cbf6905e (BELTFIX3 fixes 1-4 + HIREFIX) and one small bot fix. Follows docs/research/beltfix2-analysis.md.

Tags: V run = counted from the run data (`result/all-minutes.log` and `logs/bot-0.log` of gate1-20261103-0053 on the worker host, read only; the earlier runs from the minute archives of branches agent/gate1run-20261102 and agent/gate1run-20261103 at 174e9e68); V read = read in the code at origin/main ccd1cc5f; I = inference; U = unverified.

## 1. BELT: the 7-minute window is poverty at character creation, not a defect

The d2dsoak rule replayed over the 1239 minute lines gives one window of 7 (lines 7-13, 00:54:19-00:54:44 wall): clvl 2, Blood Moor, gold 10 to 16, hp min 23 percent, step `Recovery: farming gold`. V run. Everything else is 3 or less (list below).

- The three starting Minor Healing potions were drunk in the first 3 virtual minutes at clvl 1 (lines 0-3), the character had 0 gold, `poverty mode: on` (gold 0 below kit cost 190). Town visit at line 5 (emergency portal): `shop: visit bought {} broke true gold 0`; a sell gave 10 gold; a Minor Healing costs 30. Nothing could be bought. V run.
- The two earlier runs have the identical first 16 lines (same seed, same moves, same gold). In both a Minor Healing potion dropped at line 8 (belt count 1), which split the stretch into windows of 2 and 3. In this run one kill fewer by line 8 (kills 42 against 43), no drop, the stretch ran to line 13 when a Rejuvenation dropped. V run.
- Classification: poverty at the limit, decided by one potion drop. Not "trip not triggered / trip failed / belt placement / potion not drunk" (belt placement and the town paths ran: every `Entered Rogue Encampment` with healing 0 and gold >= 30 is followed by a shop line except the case in section 2). The judge counted correctly (my replay equals d2dsoak's 7). A bot cannot fix this window; the only levers are the gate (a minimum character level or gold for the belt rule: spec question) or luck. Ranked fix #5 of beltfix2 (mana-only kit trip must not spend the potion gold on a scroll) did not fire anywhere in this run (the clvl 6 kit trips at 01:02-01:04 had 4 to 6 healing potions), so it stays unimplemented. U whether it will matter in another run.
- BELTFIX3 paths ran (V run): `town trip` with emergency true after arrival (e.g. 00:54:17, 01:44:14 gold 462), `walking the level chain` 46 times, `back to field: portal in town` 175 times of which 19 with attempt 2 (the portal retry), 30 `"emergency":true` trips. `waypoint to town failed` count 0 (62 in the second run). The 37-minute Dark Wood window did not recur.

Other windows: 2 (00:54:10, clvl 1, gold 0), 3 (01:04:11, clvl 6, section 2), 2 (01:44:14, clvl 12, gold 462: the emergency trip ran). V run.

## 2. The real near miss: the 3-minute window at 01:04 (a bot defect, fixed)

Line sequence (V run): 01:04:01 trip in Cold Plains area, 4 healing and 1 mana in the belt, bought 1 mana; 01:04:06-08 out of mana at hostiles 42-43, four healing potions drunk in 6 s, `chose chicken first life=21%`, Save and Exit; 01:04:10 `Entered Rogue Encampment`, gold 185, healing 0, then `recovery: kit gate on`, `poverty mode: on (gold 185 below kit cost 190, 0 healing potions)`, `poverty advice play-lean`, `wp: used 1 -> 3`, back out. No `town trip`, no `shop:` line. The next potion came at 01:04:21. Window 3 virtual minutes at hp min 21 percent.

Cause (V read, internal/bot/goal.go `choresDue`): `broke := recoveryActive() && gold < RecoveryKitGold (500)` switched the generic `N healing potions` chore off entirely. The comment gives the reason (an empty purse buys nothing, G1 attempt 9), but the gold test is `p.Gold >= minPotionPrice (30)` and the shop is capped by `potionFloor` (2 potions while recovering), so in town the visit is free and buys at most the floor. This is the same shape as BELT A of the second run (town visit, 0 potions, gold, left unshopped), one branch that BELTFIX3 #1 did not cover.

Fix (commit below): `(!broke || (p.InTown() && healing == 0))`. In the field recovery behaves as before; with a healing potion in hand nothing changes. Tests (fail when reverted, V: they failed before the edit): `TestRecoveryTownArrivalRestocksHealing` (185 and 201 gold, recovery on, in town: emergency + potions; potionFloor stays 2), `TestRecoveryStillQuietWithPotionOrNoGold` (20 gold: no chore). `TestScenario_KitlessClvl7PackDeath` (gate-off subtest) expected a recovering town character with 35 gold to walk out unshopped; its gold is now 29 (below one potion) with a comment, because 35 gold now buys a potion first. Replay of the window = the first test (rows of the minute lines: gold 185 and 201, area 1, belt empty).

## 3. XP/h: 8162 against 9172 (and 10042, 9754)

Same seed and character in all three runs, three different bot commits (8c7d657, cd4c7cb, cbf6905), one fault each at 60 wall minutes. V run for every number.

| run | vmin | final xp | XP/h | kills/h | fault XP loss | clvl at vmin 100 / 300 / 500 / 700 / 1000 | town area share | state recovery (vmin) |
|---|---|---|---|---|---|---|---|---|
| 1939 (8c7d657) | 1182 | 197999 | 10042 | 375 | 0 | 6 / 11 / 12 / 14 / 15 | 10.2% | 424 |
| 2210 (cd4c7cb) | 1147 | 186684 | 9754 | 373 | 280 | 6 / 10 / 12 / 13 / 14 | 8.7% | 202 |
| 0053 (cbf6905) | 1239 | 168717 | 8162 | 330 | 7730 | 5 / 10 / 11 / 12 / 13 | 11.3% | 427 |

Where the gap comes from (against the mean of the two earlier runs, 9898 XP/h, gap 1736 XP/h = 35.8k XP over 20.65 vh):
1. The injected kill -9 cost 7730 XP in this run (xpRisk 7197 at the last save, restored to 102317 at line 706) against 0 and 280: 374 XP/h, 22 percent of the gap. It is when the last d2d save happened relative to the fixed fault time, not bot behaviour. Without it: 8545 (-6.8 percent against 9172, inside the tolerance 1834; the lower edge is 7338).
2. Early game: XP at vmin 100 was 11.4k against 19.4k and 15.0k. Lines 0-15 are identical in the two earlier runs and differ by single kills from line 6 in this one. At line 16 the 2210 run sold loot for 450 gold in town (C: 22 gold), at line 6 and 8 differ by single kills, and the gold and potion supply of the first 60 minutes diverged from there (clvl 4 at vmin 47 against 37, clvl 5 at 90 against 62). I: loot luck of a divergent seed path, not a code effect. First 500 vmin: 10.3k XP/h against 11.9k and 10.8k.
3. After vmin 500 the rate was 125 XP/vmin (137 without the fault loss) against 151 and 149. At cl12+ in the field 142 XP/vmin against 171 and 150; recovery minutes at cl12+ were 202 against 186 and 0 and cost about 64 against 145 in the field: the run with 0 recovery minutes (9754) is not above the one with 186 (10042), so recovery length alone does not decide. I: C lagged one to two levels in the whole second half, and the planner picks the area by level, which compounds the early lag.
4. Death: one, at line 316 (clvl 10, Crypt, hp 25 percent then 0), XP lost 0, back at the death spot two wall seconds later (about 2 vmin). Negligible. 18 restarts are save points (fresh games), none a crash.
5. Town time (the walk-home chain, shop on arrival): the town area share is 11.3 percent against 10.2 and 8.7, the `town chores` state 12.8 against 11.6 and 9.4. Upper bound of the cost: the 1.8 to 2.6 points of time are 150 to 250 XP/h, i.e. 9 to 14 percent of the gap, and part of that is the larger belt of a longer run. 192 `town trip` lines in 20.65 vh (9 per hour).

Share (I, three runs with different code, one seed): fault luck 22 percent, early-game loot divergence about half, town/shopping at most 10 to 15 percent, the rest noise. The baseline itself spans 6600 to 11743 (2 runs), so 8162 and 8545 are inside the baseline's observed range; d2dsoak prints INCONCLUSIVE because it needs 5 seeds. Evidence that the shopping or walk change costs XP: none beyond the 1.8-point town share. Remedy proposal: none now; if the town share must go down, the cheapest lever is a longer gap between non-emergency kit trips at low gold (`KitTripCap` and the trip budget already bound them), not belt protection.

## 4. Gate 1 re-run

Worth it after this fix, but only as a multi-seed run: one seed cannot decide XP/h (PASS/FAIL needs 5 seeds per d2dsoak) and the BELT limit is hit by one potion drop at the start of this seed. Facts for the user (spec question, not decided here): (a) whether "at or above the baseline" means within tolerance when d2dsoak says INCONCLUSIVE; (b) whether the belt rule should start counting at clvl >= 3 or when gold >= one potion, since the only failing window is a fresh clvl 1-2 character with 0 to 16 gold. I proposed the sharper form of (b) already in beltfix2-analysis.md section 7.

## 5. Not verified live

Both tests are offline (`go test -p 1 ./internal/bot`). Live check: a recovery-mode character that chickens with 0 healing and 100-500 gold must show `town trip ... emergency true` and `shop: bought ... Healing` (at most RecoveryPotionFloor) before `wp: used`.
