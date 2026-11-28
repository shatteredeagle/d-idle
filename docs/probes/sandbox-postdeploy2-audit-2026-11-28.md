# Sandbox post-deploy audit 2, 2026-11-28 (Doofer 0.0.261128020401, Questy 0.0.261128020453)

> status: research, read-only (card postdeploy2). Follows `docs/probes/sandbox-postdeploy-audit-2026-11-27.md` (the "earlier audit"). Target classic 1.14d offline sandboxes, local docker. Only `d2bot ops log --tail`, `ops status --why`, `ops quest status [--why]` were run; nothing written, restarted or flipped, no token printed. Times EST (Z minus 5 h).
>
> Tags: **V** read in the bot log; **S** arithmetic on V lines; **U** unproven. Cause classes A to K: `docs/roadmap-fable.md` ("Operating rule").
>
> LIMIT: the new builds started 21:04:16 (Doofer) and 21:05:05 (Questy); the log was read at 21:32, so the "after" window is only about 26 minutes (0.43 h, 27 minute lines each), not 40. Rates are noisy; verdicts say so. The "before" window is 18:00 to 21:04 (3.04 h, builds ...225902 and ...225945).

## 1. Health, before and after

| | Doofer before | Doofer after | Questy before | Questy after |
|---|---|---|---|---|
| clvl | 17 to 18 | 18 | 16 | 16 to 17 |
| xp/h (S, minute lines) | 18.5k | 10.3k | 8.2k (earlier audit 5.8k) | 18.9k |
| deaths (minute `deaths`) | 0 | 0 | 0 | 0 (V; the `death report` line at 21:05:02 shows 24 lifetime, `lastHour` 0) |
| ERROR/panic | 0 | 0 | 0 | 0 |
| WARN classes after | | `area skipped`/`leveling target not reachable` x6, `action failed` x3 (Countess), Save and Exit x2, `xp lost` x1 | | `area skipped`/`not reachable` x14, `points not spent` x1, `open portal did not lead to town` x1, deploy restart d2d WARNs |

Doofer's after-rate is dragged by 3 failed Countess tries in Tower Cellar 3 (21:06 to 21:07) and a farm reset; it is not a regression of the fixes (U).

## 2. Verdict per fix

| # | fix | verdict | evidence |
|---|---|---|---|
| 1 | zonepace (Questy zone at clvl 15-16) | **works** (short window) | V before: Dark Wood `zone choice ... xp/min 90` 18:00 to 20:07 (clvl 16, 5.8k to 8k xp/h), Black Marsh from 20:11 (xp/min 121 to 189). V after: `21:05:51 zone choice: Jail Level 1 margin 0.4 (need 1.5; low 0.3) xp/min 959 why: hole_first_field`, then `21:08:52 Catacombs 3 / 2` (both `area skipped ... unreachable: 10 hops and no activated waypoint`), `Tower Cellar Level 1 margin 1.1 xp/min 252 pays_for_risk` and from `21:12:30 Tamoe Highland margin 2.2 xp/min 477 best_pay_eligible`. Questy xp/h 18.9k (S) against 5.8k/8.2k. See problem 1: the picker proposes unsafe holes it cannot reach. |
| 2 | auraflap | **works** | V `combat aura up` on Questy: 7 in 0.42 h after (21:06:04 Holy Fire, 21:13:46 Might, 21:15:00 Holy Fire, 21:16:11 Might, 21:16:26 Holy Fire, 21:20:54 Might, 21:21:01 Holy Fire) = about 17/h; before 459 in 3.04 h = 151/h (earlier audit 190 per 70 min). Still alternates Holy Fire/Might, now 1 to 8 minutes apart, mostly at fight start (U: two deciders remain, rarely disagreeing). |
| 3 | relaunches | **works on Doofer, not exercised on Questy** | V Doofer `21:27:56 relaunch reason=savepoint cause=level=18` and `21:27:58 relaunch reason=newgame cause=from=farmreset.go:291`; minute line `relaunches=2 relaunchBy=newgame=1,savepoint=1`. Questy: no relaunch in the window, minute lines carry no `relaunches` key (key absent at 0, U: zero-omit). The `Save and Exit (chicken / leave game)` WARN text is still the same for both (2 WARNs on Doofer), so the reason is only in the new `relaunch` line. |
| 4 | recoverydeath | not exercised | V 0 `zone recover` lines after; before 2 (`reason=potion_crisis via=return` on Questy, clvl 16). 0 deaths. |
| 5 | mercgold | not exercised | V Questy `merc: alive` since the deploy (route guard now `merc=alive`, floors 10/14/10; before `merc=none`, floors 10/15/11); 0 `merc: none at clvl` and 0 `merc dead since` lines after on either. Doofer merc alive (`seen class=271 alive=true`). The dead-merc revive trip: before, Doofer revived 22 min after the death although stash gold was 98k (V, earlier audit); no new death to test. |
| 6 | fieldstate hand-over | not exercised | V 0 `handing over to the fighter` after (1 before, Questy WARN `back to field: hostiles near on the way, handing over to the fighter`). |
| 7 | refusalPark / tuningwrite | as expected | V 0 refusal parks; 0 tuning-damage WARNs. |

## 3. Questy auto-start (8)

V (EST): `20:01:24 quest auto-start counting down`, `20:04:44 sisters_burial_grounds/blood_raven started by itself`, `step Go to the Burial Grounds`, `20:06:04 step Kill Blood Raven`, `20:07:11 hand-in queued`, `20:07:13 hand-in served`, `20:07:26 handed in` (akarafix hand-in works, first real use). `20:44:22 countdown`, `20:45:00 forgotten_tower/countess started by itself (away=true)`, `20:46:31 step Kill The Countess`, `20:47:29 auto-finish ... the quest reads done, the stale run was closed`. So both chunks finished, 0 deaths, 0 parks, no `paused_after_deaths/no_path/refused_repeatedly` code. Blood Raven also gave the merc (the hero was `merc=none` at 18:00 and has one at 21:05). Now: `sisters_burial_grounds` and `countess` done, `tools_of_the_trade/malus` ready auto=on, `search_for_cain/scroll` ready auto=never, andariel `available auto=never need=margin_wall word=too_hard` (V 21:32 `quest status --why`). Malus: `21:17:31 quest auto-start counting down ends 21:17:51`, then no `accepted`, no `started by itself` line and no refusal in 15 minutes (V); the hero stays in Tamoe Highland. Why is U (the hero was mid-fight at 81 % life, then a town trip at 21:20; auto-start may need the step to be idle) .

Doofer: Countess parked a third time, `21:07:42 parked forgotten_tower/countess: gave up after 3 tries ... gate to level 24 ... not reachable from {42 20} (cause=cut_off)`, after 3 tries in 30 s (21:06:58, 21:07:11, 21:07:25). Same cut-off cell as the earlier audit.

Jail Level 1 choice (clvl 16, margin 0.4): V the hero left town 21:05:55, reached Monastery Gate and Tamoe Highland, retreated to town at 21:09:51 with life 91/231 and 3 potions (`zone entry area 1 life 91/231`); 0 deaths. It never entered Jail. So the choice was not tested against Jail and the survival was the Highland fights on the way (U on whether the margin is right).

## 4. New problems, ranked

1. **Zone picker offers holes it cannot reach and repeats them** (class G/D). V Questy `21:08:52 Catacombs 3, Catacombs 2: area skipped unreachable ("no activated waypoint on the way")` x14 in 26 min; Doofer the same `Catacombs Level 3 margin 0.1 hole_first_field` every 3 minutes (21:07:47, 21:13:24, 21:16:32, 21:20:00, 21:23:02, 21:27:19) then falls back to Tamoe. Cost: a replan per cycle, noise; risk: margin 0.0 to 0.4 picks for a clvl 16 to 18 hero. Card `holereach`: drop an area from the candidates for the session once it is `unreachable`, and do not choose a `hole_first_field` below `low 0.5` while a safe zone pays above 200 xp/min.
2. **Countess retry thrash on Doofer** (class D/B). Third park for the same cut-off gate; the retry still burns 3 tries in 30 s. Carries over card `parkreason` from the earlier audit (retry only when map or position class changes), plus the Tower Cellar 3 to 4 nav fixture.
3. **`xp lost lost=368946 before=368946 after=0` after a savepoint relaunch** (class K). V Doofer 21:27:58, one WARN directly after `xp at risk why=save point`; the next minute line has xp 368957, so the hero lost nothing. A reader or counter sums 369k lost xp. Card `xplostfalse`: the xp-loss detector ignores a read of 0 right after a relaunch.
4. **Questy Malus auto-start never fires** (class B, U). Countdown ended 21:17:51 without accepted/refused line for 15 min. Card `autostartsilent`: when a countdown ends and nothing starts, print the reason code (the same word the board uses for refused).
5. **Aura still alternates Holy Fire/Might** (class A, minor). 7 swaps in 26 min. Planner shadow `agree` still ignores aura (earlier audit problem 8). Card `plannershadow-aura`.
6. **`points not spent attempt=1 stat=5 skill=1`** (class K, one WARN, Questy 21:08:52 right after the level-up); the leveler spent them at 21:09:52 (`leveler: done`). One-minute lag; watch only.
7. **`Save and Exit (chicken / leave game)` WARN still shared by resets and chickens** (class K). The new `relaunch reason=` line fixes the reading, but the WARN text should carry the reason. Fold into `leavereason`.
8. **Doofer stash gold unused, 104.9k at 21:31** (V minute `stashGold`), kit: potions 6 and `[kit: healing potions 3/4]` (class C, unchanged from earlier audit; card `goldsink`).

## 5. Not checked

Planner shadow lines, admission counters (V `admission agree 249 differ 0` on Questy, shadow), `combat aim/stance/minion summary`: not re-run. leakcheck: see commit message.
