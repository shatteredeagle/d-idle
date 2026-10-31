# Soaky soak analysis (SOAKANALYSIS, 2026-10-29)

> status: SUPERSEDED in part: pre-fix baseline; the approved death rule is plan-v2 section 10.2 and openspec/specs/death-rule/spec.md.

Where the time of a fresh no-cheat Sorceress went over 3.5 hours, what it cost, and what to fix first. Offline analysis of
the bot logs and soak samples; no game, no service touched. Tags: **V** = counted or read by me from the files named (method
in "Data"), **I** = inference from V (reasoning stated), **U** = unverified guess or from memory.

## Data

Character Soaky (Sorceress, Normal, offline, i4, never cheated; fresh clvl 1 at 07:41). Four bot runs, binary replaced between them
(`livebot/d2bot` mtime 10:48, so only the last run is the newest build; the earlier three ran older builds, U which commit):

| run | window | clvl | how it ended (V, last log lines) |
|---|---|---|---|
| 074137 | 07:41:42 to 08:52:23 (71 min) | 1 -> 4 | `recycle: failsafe restart`, `leave failed (read 4 bytes ...: EOF)`: stopped after a 25 min pause flap |
| 085652 | 08:56:57 to 09:16:25 (19 min) | 4 | `shutting down: finishing in town` (graceful, a redeploy) |
| 091641 | 09:16:48 to 10:44:02 (87 min) | 4 -> 6 | Go panic, nil deref in `town.(*Town).walkToTown` (trip.go:114) called from the boss escape |
| 104840 | 10:48:47 to 11:12:44 (24 min) | 6 -> 7 | log cut while `recovering` at life 2 (still running when I fetched it, or killed) |

Sources: `~/d2lod-live/livebot/runs/20261029-{074137,085652,091641,104840}-d2bot.log` (JSON slog) and
`~/d2lod-live/soak/i4-20261029-{074315,085700,091750,104844}/samples.jsonl` (30 s samples) on the worker host, copied to a scratchpad and parsed
with throwaway Python. Time totals come from `state` transitions; area time from `Entered <area>` events (time in the area up to
the next entry, the final area of each run is dropped); XP from the `xp` field of `combat kill` lines. Caveats:
- 12 to 15 percent of `combat kill` lines have `xp:0` because one kill line carries a batch (e.g. Fallen `0,0,0,90`): per-kill XP is
  unreliable, per-area sums are fine (V: the sums equal the sample counter deltas, e.g. run 085652 2728 = 2728).
- The soak sample counters `xp/kills/deaths` are cumulative over the bot's life (21605 at Soaky's first sample), only deltas mean anything.
- In-level XP (`charXp`) is only in the newest samples; clvl XP thresholds below are the classic table (U: recalled, but the
  clvl 6 to 7 window in run 104840 matches it to 83 XP, V).
- The status log has no gold, belt or inventory fields, so gold flow is only visible through poverty, shop and sell events.

## 1. Where the time went (V, 200.7 min of logged time, wall 211 min)

| bucket | min | share | notes |
|---|---|---|---|
| fight (state `clearing`, includes kiting, out-of-mana backoffs, rests, flee) | 130.9 | 65.2% | |
| paused flap (one incident, run 074137) | 24.5 | 12.2% | see R4 |
| walking/doing a town trip ("Going to town" 16.5, chores 5.7, gear 2.2, quest talk 1.1, heal 0.8, points 0.7, failsafe 0.3) | 27.3 | 13.6% | 37 trips |
| travelling between areas (`traveling`) | 16.0 | 8.0% | |
| re-entering (save-and-exit, menu, death recovery) | 1.9 | 0.9% | 13 chicken exits + 1 death |
| unlogged gaps between runs | about 10 | n/a | 3 restarts |

By clvl (V, wall minutes and shares; fight / town / travel / paused): clvl 1 12 min (58/17/21/0), clvl 2 8 (89/9/0/0), clvl 3 19 (85/13/0/0),
clvl 4 60 (52/3/3/**40**), clvl 5 32 (81/16/1/0), clvl 6 65 (60/**20**/**17**/0), clvl 7 5 (76/22/0/0).
Level-up times (V): L2 07:54:00, L3 08:01:44, L4 08:20:26, L5 09:26:15, L6 09:57:52, L7 11:07:24. So 12, 8, 19, 66, 32, 70 minutes per level.

Inside "fight" (V counts, I for time): 199 `combat out of mana: backing off`, 12 `resting before the pack` (274 s planned rest in total, so mana
rests are small: about 2 percent), 89 `abort fight: flee` (75 in run 074137 alone, mostly clvl 2 to 3 in Cold Plains), 409 `nav stuck` and
108 `combat move failed` (mostly kiting in tight cave geometry, I: cheap, no area is blocked by them), 55 healing and 9 mana potions drunk.

## 2. XP per hour by area and clvl (V; area-seconds, kills, XP, XP/h)

| area | clvl | s | kills | XP | XP/h |
|---|---|---|---|---|---|
| Den of Evil | 1 | 430 | 47 | 936 | 7.8k |
| Den of Evil | 4 | 1474 | 274 | 5304 | 13.0k |
| Den of Evil | 5 | 177 | 50 | 840 | 17.1k |
| Den of Evil | 6 | 2106 | 578 | 10435 | **17.8k** |
| Cold Plains | 2 | 191 | 50 | 1335 | 25.1k (chickened out every 2 min) |
| Cold Plains | 3 | 817 | 85 | 2220 | 9.8k |
| Cold Plains | 5 | 885 | 142 | 4344 | 17.7k |
| Blood Moor | 2 / 3 / 5 | 195 / 171 / 429 | 25 / 7 / 78 | 411 / 68 / 1431 | 7.6k / 1.4k / 12.0k |
| Burial Grounds | 5 / 6 / 7 | 166 / 311 / 289 | 60 / 4 / 20 | 683 / 176 / 567 | 14.8k / 2.0k / 7.1k |
| Cave Level 1 | 6 | 71 | 7 | 440 | 22.3k (71 s only) |

Reading (I): the Den of Evil was the best sustained XP at clvl 4 to 6 (about 1000 kills/h of weak monsters, V 578 kills in 2106 s).
At clvl 6 it took 66 percent of the in-area time because the bot re-enters a fresh game after each chicken, and a fresh game has a
fresh Den. Burial Grounds is poor at every level because the XP there only comes from the Blood Raven fight, which the bot loses (R1).

Overall Soaky made 23.3k XP true (see section 6) in 211 min = 6.6k XP/h wall, 8.7k/h by the telemetry counter.

## 3. Deaths, near deaths (V)

- **1 death**: 10:00:39, Burial Grounds, clvl 6, killer Hungry Dead. Chain: `0 healing potions low` emergency trip at 10:00:29 with no TP in
  the pack, three `combat town portal attempt failed: inventory panel did not open`, `town trip abandoned`, dead 10 s later. Corpse
  retrieved (`corpse attempt ok`).
- **13 chicken exits** (save-and-exit at 12 to 57 percent life): 6 at clvl 2 to 3 in Cold Plains within 14 minutes (07:58 to 08:12), 5 in run
  091641, 2 in run 104840. Cause chain each time: potions 0 or 1 (belt empty), `abort fight: flee: no potion, no portal, life falling`.
- **Life stuck at 12 to 26 percent for minutes** in 75 of 141 status lines of run 074137 (half of them are the 25 min pause) and 9 lines
  09:11 to 09:15 of run 085652: the bot keeps farming the Den at about 20 percent life (`cannot regenerate`, no potions).
- The last act of run 104840 (11:12:31 to 11:12:44): 3 Blood Raven attempts, life 16 to 19 percent, `boss: escaped to town`, then exit at life 2.
- Run end 10:44: nil-pointer panic in the walk-home path (fixed since: `a305319 town: ToTown no longer dereferences a nil snapshot ...`, V in git log).

## 4. Repeated failure patterns (V counts)

| pattern | count | cost |
|---|---|---|
| `0 healing potions` emergency trips | 11 of 37 trips (plus 12 "heal below the resume line" after a chicken) | each 15 to 83 s |
| no town portal: walking or waypoint home | 10 trips (walk 5, waypoint 5; only 2 trips used a TP) | 30 to 83 s per trip, then `town: no way back to level 8` twice per Den return (`back to field` falls back to travelling, V) |
| leveler equip retry: `Amulet 8.0 -> neck: item N did not reach slot neck` (2 items x 2 tries each town visit), `Ring -> rHand` once | 8 failures in 3 runs | about 17 s per visit, same swap retried every visit |
| `combat town portal attempt failed` (panel did not open / no portal appeared) | 6 | caused the death and one forced exit |
| `town chore failed` (shop: NPC greeting never gave way to the menu; repair: pointer never hovered) | 2 | |
| Den of Evil `ran but is still needed (evidence 2)`, `blocked` | 5 times | Corpsefire killed twice (10:28:59, 10:57:26) and the save still reads `den_of_evil=0x001c` both before and after (V). No Akara turn-in was ever logged after a clear (the only Akara talk is at the start of run 1, which itself failed once: `pointer never hovered the target`) |
| Blood Raven `boss run` aborted `context canceled` | 181 in run 091641 (1 s each, V sum 177 s), 8 in run 104840 (330 s, 0 kills) | See R1 and R5 |
| Burial Grounds / Cold Plains border ping-pong | about 40 `Entered` events 09:50 to 09:56 (V) | the fight/dodge/kite crosses the border; I: the boss run and `level.6` fight over the area |
| `before-save guard failed: cheat: character not on the allowlist: "Soaky"` | 5 | harmless noise before each save-and-exit (the exit still ran), I |

Gold flow (V, partial): poverty mode on 07:53 (gold 10) and 09:25 (gold 22); poverty sell trips at 44, 83 and 22 gold; one full-pack trip
(10:57, 195 s) sold items for a 2399 gold budget; the gear shop then spent **1175 gold in one visit** (11:00): Javelin 240, Cap 47, Sash 60, Leather
Gloves 49, Heavy Gloves 330, Spear 281, Short Staff 168 (`estimate` for the Javelin was 5, actual 240; the Short Staff immediately replaced
the Javelin, the Spear was never equipped, I: about 520 gold wasted). Five minutes later (11:09, 11:11) the belt was empty again
(`0 healing potions` x3). No log line says what the `shop` chore bought (no potion purchase is logged), so whether potions were bought at
11:06 and drunk within minutes (17 potions in 24 min, V) or not bought is U.

## 5. Comparison with LEVELRATE (docs/leveler.md, Booti at clvl 12, 2 h)

| topic | LEVELRATE (Booti) | Soaky now |
|---|---|---|
| XP rate | about 3.5k/h farming, 4.0k total in 2 h | 6.6k/h wall true (clvl 1 to 7), levels are cheap so not comparable per level |
| area choice | stuck: no quest, Pick refused everything, fell back to Den/Blood Moor (40-70 XP/h) | the campaign runs (Den, Burial Grounds quest), and Den gives 13 to 18k XP/h at clvl 4 to 6 (V); the Floor fix (`worthFloor`) was not exercised (clvl too low) |
| waypoints | none, so no hops | Cold Plains waypoint used 5 times for the trip home (`no town portal: taking the waypoint to town`, V): improved |
| mana | 0 energy, 30 to 40 s rest per pack | 2 of 5 points into energy from clvl 5 (V plans `+1 energy ...`); `out of mana` backoffs 138 in run 091641 (87 min), **0 in run 104840 (24 min)**, 3 rests (V). I: energy plus a staff help; U how much each |
| poverty | 12 heal + 10 empty-belt trips, 0 gold, no TP | unchanged: 11 empty-belt trips, 12 post-chicken heal trips, poverty on twice. `poverty advice play-lean` is wired and fires once (V 09:25:51) but did not stop the loop |
| re-entry resets `cleared` | 16 re-entries | 13 chicken exits; for the Den this is a feature (fresh monsters) |
| walking home without scroll | not possible | works for 8, 3, 17, 9 (hopToNeighbour, V use at 09:23 and 10:57), and a crash in it was fixed |

Not improved: nobody buys a TP (V: 3 TP casts in 3.3 h), the belt is empty most of the time, boss fights are attempted regardless of potions.

## 6. Progress counter vs character XP (open question)

The telemetry counters add up to 30.6k XP (07:43 to 11:12). The character data says Soaky has 23.3k: `charXp 5827` at clvl 6 plus the 14175 to
reach it at 10:49 (U table), +3.3k by 11:12. About 7.3k XP (24 percent) counted but not kept (V arithmetic, table U).
Candidates (I): unsaved play lost when a run is killed without save-and-exit. The XP earned after the last save-and-exit was 2619 (run
074137, ended `failsafe restart`, leave failed) and 2900 (run 091641, ended by the panic): 5.5k of the 7.3k. Run 085652 ended gracefully (finishing in town) and run 104840
shows no loss (exact match). 1.8k stays unexplained (1.2k inside run 091641 between clvl 5 and 6 where nothing restarted).
`internal/launch/recycle.go:59` assumes the character is saved on every area change; `cmd/d2bot/live.go:289` says the .d2s is written only on Save and Exit.
These two comments disagree and this data can only be explained if the first is not reliable. Test (needs live, no code): log `charXp` in the 30 s
status line (already in samples) and compare before/after the next hard restart.

## 7. Ranked recommendations

**R1. Stop fighting Blood Raven underlevelled and without supplies (highest value).** Evidence: 8 + 181 attempts, 0 kills in two runs; clvl 6 to 7
attempts at danger 2 cost 11 healing potions in 358 s of fights for 20 kills (V `combat fight area=17 danger=2` summary, run 104840) and the
5 attempts drained the belt to 0 three times in 6 minutes (11:06, 11:09, 11:11); the only death was there; area XP at clvl 6 is 2.0k/h against
17.8k/h in the Den. The planner says `Burial Grounds: Blood Raven (Rogue quest, good XP)` at clvl 3 and 5 (V 08:19:40, 09:49:34). Fix: gate the boss
step on clvl (U: 8 or higher), on at least 4 potions plus a TP, and on mana; when the gate fails keep farming the Den. Code: planner text in
`internal/bot/leveling.go` / `plandata`, `internal/task` `StepKill` preconditions (commit 375cc93 already gates Smith and Countess on potions, copy the pattern),
`taskActs.KillBoss` in `internal/bot/task.go:473`, `internal/boss/run.go` (respond/escape).

**R2. Fix the potion and TP supply loop before anything else buys gear.** Evidence: section 4 gold paragraph; 55 potions in 3.3 h; 11 empty-belt
trips; 10 trips without a TP (5 forced walks, 30 to 83 s plus the walk back). Fix: in the gear-shop budget keep a reserve (kit cost = 1 TP +
HPFirst potions, `resource.KitCost`) and buy the kit first; fix the price estimate (`internal/leveler/score.go estimatePrice` returned 5 for a 240 gold Javelin, so the
shop spends and then cannot afford potions) and do not buy off-class weapons (Javelin, Spear for a Sorceress, score 1.2 and 3.6) when a staff is available; log what the `shop` chore buys.
Code: `internal/leveler` shop/buy loop, `internal/resource` (`BuyKit`, `PotionGoldPct`, `ReserveGold`), `internal/town/chores.go`.

**R3. Keep a low-level Sorceress out of Cold Plains until it can survive (clvl 2 to 3).** Evidence: 6 chicken exits in 14 min at clvl 2 to 3 there (V),
each 20 to 35 s of re-entry plus lost position; planner reason `Cold Plains WP; Burial Grounds 17 for Blood Raven` picks it at clvl 2.
Den (7.8k/h at clvl 1) and Blood Moor (7.6k/h at clvl 2) did not chicken. Fix: minimum clvl or "potions >= 3" for Cold Plains in the planner; I: gain is
fewer exits and about 10 min on L2 to L4, not more XP/h (Cold Plains pays 10 to 25k/h while it survives).
Code: `internal/bot/leveling.go` planner (`Plan.Pick`/ area tables in `plandata`).

**R4. The 25 minute pause flap (12 percent of the whole soak).** Evidence: 08:27:13 `failsafe: paused` (`Life 26% in area 17, no potion, and the way to town failed
3 times`, `heal trip: town: no town portal available` x3), then 08:27:17 to 08:52 `paused <-> clearing` every 2.5 s (V, 25 min, 0 XP), ended by `failsafe restart`
with a failed leave. Since then: `hopToNeighbour` has 17 (`f5b7197`) and the walk-home path was fixed, so the cause may be gone, U. Needs: a test that a
self-paused runner does not flip to `clearing` on each tick (`internal/bot/loop.go` `selfPause` / `levelerResume` / `autoResumed`, lines 94 to 100 and 573 to 600); the flap itself is a bug
regardless of what caused the pause.

**R5. Boss-run vs level step thrash and the border ping-pong.** Evidence: 181 aborted `boss run`s of about 1 s in 87 min, `level.6` step every 3 s, 40 Cold Plains /
Burial Grounds entries in 6 min (V), `run finished blood_raven` printed at 09:56:36 although BR was never killed. I: the planner hands the
boss run to the executor and `LevelTo` cancels it each tick. Code: `internal/task/executor.go` Tick/`LevelTo` (`internal/bot/task.go` `taskStep`, `freshGameToLevel`),
`internal/boss/run.go`. Lower value than R1 once R1 stops the attempts, but it also hides real progress in the logs.

**R6. Den of Evil quest never completes, reward not collected.** Evidence: `ran but is still needed (evidence 2)` x5, quest word unchanged after killing Corpsefire; no
Akara turn-in step after the clear. U: the Den reward (a skill point; I recall classic Normal gives one) and the word `0x001c` meaning "not yet completed"; the task graph marks
the step `Skip: true`, `Tag: "U (reward claim by re-talking Akara)"` (V `internal/task/act1.go:46`). Do: add the Akara return after a verified clear (all monsters dead
in the same game, then talk to Akara); needs a live check on Soaky (no cheat needed). Code: `internal/task/act1.go`, `internal/quest`, `internal/town` (Akara dialog,
the click `pointer never hovered the target` failure at 07:42 is the same dialog).

**R7. Stop retrying equips that cannot work.** Evidence: `item N did not reach slot neck` on every town visit (8 failures, about 17 s each, V). Do: remember failed
(item, slot) for the session and skip; find why the amulet does not reach `neck` (U: requirement, or the click lands off the slot). Code: `internal/leveler/equip.go:225-266`.

**R8. Save before every planned stop and consider periodic save-and-exit.** Evidence: section 6 (about 5.5k XP lost to two hard stops, U causal). Code: `cmd/d2bot/live.go`
shutdown sequence (already saves on SIGTERM), `internal/launch/recycle.go`, `tools/live.sh`/the redeploy path (use SIGTERM and wait, never `kill -9`; the run 074137 exit was a failsafe restart whose leave failed).

## 8. Work packages for d2-worker agents (offline, testable)

| id | package | acceptance | dirs |
|---|---|---|---|
| WP1 | Boss gate: Blood Raven (and any danger-2 boss) requires clvl, potions >= N, TP, mana; else `blocked: not ready` and the planner falls to the best farm step. Table-driven test with the exact numbers from the 11:05 to 11:12 log (belt 0, clvl 6) | gate refuses; with kit and clvl 8 it allows; planner picks Den | `internal/task`, `internal/bot` |
| WP2 | Gear-shop budget keeps a kit reserve, kit bought first; `estimatePrice` regression test with the Javelin (cost 240); skip weapons the class cannot use when a better weapon is on offer; log purchases of the `shop` chore | test: budget 2399 buys potions and TP before the Spear; no weapon purchase for a Sorceress below the staff score | `internal/leveler`, `internal/resource`, `internal/town` |
| WP3 | Planner: Cold Plains needs clvl >= 4 or a full belt; Den/Blood Moor between | `Plan.Pick` table test for clvl 1 to 4 | `internal/bot`, `internal/plandata` (ask owner) |
| WP4 | Self-pause flap: runner with `paused=true` and no `Resume` stays in `paused`; test with a fake clock replaying the 08:27 sequence (heal trip fails 3 times) | no `paused -> clearing` toggles for 2 minutes of fake time; auto-resume after `selfPauseDelay` | `internal/bot` |
| WP5 | Boss run thrash: a boss run owns the step until it ends or its own deadline; `level.*` must not cancel it each tick; log a reason when it is cancelled | replay of 181 aborted runs gives at most one start per 30 s | `internal/task`, `internal/bot`, `internal/boss` |
| WP6 | Equip failure memory (per item and slot, cleared on level up or item change) | second town visit does not retry the amulet | `internal/leveler` |
| WP7 | Den quest closure: after a verified clear, `KTalkNPC` Akara to claim; fixture from the logged state (`den_of_evil=0x001c` before/after) | step order test; live check by d2-live | `internal/task`, `internal/quest` |
| WP8 | Status line adds gold, belt counts (HP/MP/RV), TP count, `charXp`, clvl; telemetry soak parser reads them; add per-phase time (fight/town/travel/paused/rest) to the minute summary so this analysis needs no log parsing | `d2soak` report prints XP/h per area and phase table | `internal/telemetry`, `cmd/d2soak`, `internal/bot` |

## 9. Open questions

1. Why does the 7.3k XP counter excess exist (section 6)? Live test described there.
2. Did the pause flap survive the later fixes (R4)? Replay test (WP4) answers it offline.
3. Did the `shop` chore buy potions at 11:06, and where did 17 potions in 24 min go (all drunk in the Blood Raven fights per V `combat fight` lines: 11 of 13 there)?
4. Is `0x001c` the Den "reward pending" state? Read the quest bits of a character that has turned it in (offline save compare, no game needed).
5. The newest build (run 104840) has a different fight summary (`combat fight area=... danger=... potions=...`) but only 24 minutes of data; rerun this analysis after R1 to R3 land
   with a 2 h run (the same Python approach, not committed because the logs are the source of truth; WP8 makes it a built-in report).

## 6b. XP accounting after XPLOSS (code, offline; the live part is still open)

- The `/stats` XP total is gross: `observeKills` adds every positive memory-exp step and ignores drops, so XP lost on a reload and earned again counts twice (V from code). Memory exp can only fall when the game is reloaded from the .d2s (U: death penalty is not known to exist in LoD 1.14d).
- The bot now logs `xp lost` (lost, before, after) when exp falls, `xp at risk` (memory exp minus saved exp) before rung 4 and before restarting a lost game, and `xpRisk` / `xpLost` in the minute line. Saved exp is the exp at game entry unless `Deps.SavedExp` reads the .d2s (not wired in cmd/d2bot yet).
- Save and Exit order (V from code): a planned recycle and rung 4 both run `Recycler.Recycle`, whose first step is `Leave` (Save and Exit); a failure is logged and the stop goes on. A lost game (crash, hang, the process gone) cannot be saved: `gameLost` just restarts.
- d2soak: `-save-file PATH.d2s` adds `saveXp` to every sample. The report's XP and XP/h come from saved exp (else memory exp), as a running maximum; `xpCounter*` keeps the gross value; `xp_lost` is a warning finding with the lost total. Alert `xp_at_risk` (rule `-alert-xp-risk-levels`, default 1) fires when unsaved XP exceeds one level's worth.
- Live check still needed (U): whether an area change writes the .d2s. Compare `saveXp` and `charXp` across an area change in a soak run.

### 6c. SAVECHECK live measurement (2026-10-29, i5 Multi, level 30 / exp 5000000 from an earlier cheat; V)
Tool: `cmd/d2savecheck` (read-only: .d2s mtime, level, exp, gold) next to `d2cheat show` (memory); an ssh stat watcher logged mtime changes at 1 s.
- Baseline 12:11: file mtime 10-28 20:43 (untouched), file exp 5000000 gold 777 = memory.
- Run 1 (d2bot farm Blood Moor, 200 s, town -> Blood Moor and back, shop in town gold 777 -> 127, memory exp 5000005): file still unchanged at the several area changes I sampled... but a write appeared at 12:15:29 (exp 5000005, gold 127, size 976 -> 980), 52 s after the last area change into town; the trigger is NOT identified (not d2cheat show / d2savecheck, which were rerun with no write). U: cause.
- Run 2 (70 s run, area changes town -> Blood Moor 12:16:31, -> town 12:17:21, memory exp rose to 5000012): the 1 s watcher saw NO write until 12:19:23 (2 min after the last change). Area changes alone do not write the .d2s (V, 2 changes, 3 min).
- kill -9 of i5's Game.exe only, restart and login: exp 5000005 gold 127 level 30 = the file; the 7 XP earned after the last write (12 vs 5) were LOST (V). Items were not compared (U).
- Conclusion: do not assume area changes save. Treat everything since the last Save and Exit as at risk (R8 stands); the 12:15:29 write shows some other trigger exists (U).

## 10. Progress measurement codes (M0, plan-v2 section 5; code: cmd/d2soak/progress.go)

All of these read the bot log's `minute` lines (clvl, xp, gold, step, area), never the walking progress counter that ticks while the bot idles. `d2soak report` runs them whenever a bot log is known (`-bot-log` or the run's `botlog.txt`) and merges the findings into the verdict. The death-rate check is unchanged.

| code | level | fires when | flags (defaults) |
|---|---|---|---|
| `no_gain` | fail | no clvl gain, no quest-step change, no gold above its mark and no xp gain for N min. XP gain (NOGAIN fix, V on attempt 16 fixture) = xp up by at least max(`NoGainMinXP`, 1% of the current level's xp span from the experience table) since the last progress point; a death's xp loss lowers the mark. Quest step = the name in `Quest: <kind> (<name>)`, digits folded, "Level up to N" excluded (that is farming). Gold mark = gold at the last progress point; spending lowers nothing, so only net gold above it counts. A log hole > 10 min restarts the clock (bot down is judged elsewhere). | `-no-gain 90m`; profile m1 sets 45m; `-no-gain-min-xp 500` (0 = 500, -1 = xp ignored) |
| `loop_no_gain` | fail | one (area, step text) entered N times with no clvl, quest-step, gold or xp (+200) gain in between. Entry = a change of the key between consecutive minute lines. | `-loop-repeats 20` |
| `idle_in_town` | fail | an unbroken run of `Waiting...` / `Paused` steps of at least 10 min (first waiting line to the first non-waiting line) | `-idle-in-town 10m` |
| `below_floor_clvl` | warn, fail with flag | clvl at hour 2/4/8/12/24 below 7/11/16/20/26 (docs/sme/pace-tables.md 5.2, U-derived cells). After hour 2 the floor is max(start clvl, table), so a character that begins at clvl 12 may not sit at 12 and call hour 4 fine for a floor of 11. Hours not reached are not judged. | `-fail-below-floor` |
| `state_dominant` | warn | a state group above 40% of a clock hour of the run (hours with < 30 min of lines are skipped) | `-state-share 0.4` |

Time in state: each minute line stands for the time to the next line (capped at 5 min). Step texts map by prefix (digits folded to N) to farming (`Quest: fighting (Level up...)`), quest travel (`Quest: travelling/in town`), quest fight (`Quest: fighting (Kill ...)`), town chores (`Chores in town`, `Gear:`, `Going to town`, `Returning to...`, `Settling`), recovery (`Recovery: farming`, `Game lost`, `Failsafe`, `Re-entering`), waiting (`Waiting`, `Paused`), dead-recovering (`Restocking after death`, `Retrieving the corpse`). Unknown texts go to `other` and are printed as `unmapped step text`; extend `stepGroups` when one shows up. Farming above 40% is normal while leveling, so that warning is information, not a defect.

The report also prints `clvl at N h` for every full hour (with the floor and ok/BELOW on table hours) and `xp/h clvl a-b` per band (1-5, 6-9, 10-13, ..., 26+; positive xp steps over the time spent in the band).

Fixtures (V, copied read-only from the worker host `~/d2lod-live/soak24/runs/`): `cmd/d2soak/testdata/soak_a14_minutes.jsonl` (attempt 14, 23:58-01:01, idle 00:47-00:57 and no gain from 00:36; the first 40 min contain a real 20 min stretch between gains, so tests use N=25 for both the "fires" and "quiet" cases) and `soak_a10_minutes.jsonl` (attempt 10, 40 min of "Recovery: farming gold" at clvl 10 with gold near zero: `no_gain` at 30m, not at m1's 45m, and recovery above 40%). Not yet exercised on a full 2 h+ log with deaths or on a fresh character (U).

Attempt 16 fixture `soak_a16_minutes.jsonl` (V, 03:15-05:12 of the 03:06 log: clvl 13, 111 lines, 23k XP in 2 h, one clvl-up, gold spent as fast as earned) must raise no `no_gain`; before the XP term the report said `FAIL no_gain` for 77 min (false positive). With the XP term the a10 fixture (about 4.5k XP/h while farming gold broke) no longer fires `no_gain` either: it is caught by `state_dominant` (recovery) and by `-no-gain-min-xp -1` in the test; a14's idle stays FAIL via `idle_in_town` and `no_gain` (xp flat, 17 min). The minute lines carry no xpNext, so the span is computed from `internal/telemetry` exp table (V). Limits: `no_gain` of 90m and the other defaults are U. They live in `Limits.Progress` (limits.json, so `report`, `status` judge with the run's own values; a limits.json from before M0 has none and judges nothing).

## 11. Minute line fields after M0B-LOGS (offline; V by unit tests, live values U until the next soak)

`msg=minute` keys, parsed by `cmd/d2soak/minutelines.go`:

- `potions`: healing potions in the belt slots of the snapshot (changed meaning; it used to be the drunk counter). `mana`: mana potions in the belt. Both are read from the snapshot's belt container through the item catalog (`bot.kitOf`), rejuvenation is not counted. 0 when no inventory was read (U whether a snapshot ever lacks it live).
- `potionsUsed`: the old `potions` meaning, potions that left belt or pack in the field since the bot process started (observer counter, needs the `potion` observer enabled). `d2soak` reports "potions used" from it; old logs without the key are read with `potions` as the used counter and `Potions` 0 (`Minute.HasBelt` false).
- `kills`: per-character running total, persisted in `<deathlog>.counters.json` (name + total) and reloaded when the bot restarts with the same character name; a new character name starts at 0. Game restarts do not reset it. `killsGame`: the old counter, since the bot process started.
  The 16 to 37 percent undercount against the lifetime delta was the reset at bot restart (deltas across a restart); a residual undercount from kills the observer never sees (monster dies between two snapshots, or is first seen dead, or dies in town) is possible and U until measured with the new total.
- `state`: activity group of the step text (`internal/stepgroup`, the table moved out of `cmd/d2soak/progress.go`, same texts and order), `rstate`: runner state (idle, traveling, clearing, resupplying, recovering, ...). Recovery share = lines with `state=recovery` or `dead-recovering`.

## 12. Death rule in the harness (plan-v2 10.2, approved 2026-10-30; code: cmd/d2soak/deathrule.go)

`d2soak report` judges deaths from the bot log (`character died`, `state` activity, `area_change`, `combat fight` and `minute` lines) and, when found (`-deaths FILE`, else `deaths.jsonl` in the run directory), from deaths.jsonl: follow-up lines give `secondsToField`, the autopsy gives `beltHP`, `potionsDrunk60s`, `goldCarried`. On by default in profiles m1 and m24 (`-death-rule` elsewhere); it replaces `-max-deaths-per-hour` for those two profiles (the flag still works when given). A `limits.json` without the `Death` block judges nothing new.

| code | level | fires when |
|---|---|---|
| `death_streak` | fail | 3 deaths within 30 min and no clvl gain between the first and the last (minute-line clvl), or 5 or more deaths within 60 min whatever the gain. No minimum run time. |
| `death_recovery_stuck` | fail | one death to the first `combat fight` line or non-town `area_change` above 5 min (any run age; a death not back in the field when the log ends counts up to the log end), or death-recovery time above 10 % of a rolling 60 min (fixed 60 min denominator, only once the run is 30 min old) |
| `death_rate` | fail > 15, warn > 5 | deaths per 1000 kills in the rolling 60 min (monotonic `kills`, positive deltas), only with >= 150 kills in the window and a run of >= 30 min. Windows younger than 60 min use the run start, so an early window is a smaller sample: a8 shows 13.1 at 39 min (2 deaths, 153 kills), under the limit but close |
| `death_recovery_share` | warn | recovery share above 5 % |
| `death_pair` | warn | 2 deaths in 60 min (suppressed when the streak rule failed) |
| `death_repeat_site_no_action` | warn | a repeat-site death (2 deaths in one area within 30 min, or a death within 180 s of the previous) with no log line `death: repeat-site ban` within 5 min after it. The bot side is built elsewhere; the text is only looked for |

Recovery time = the union of (a) death to back in the field, minus the time in `Recovery: farming...` or `Waiting...` steps, and (b) spans of the states `Recovering after death`, `Restocking after death`, `Retrieving the corpse` (only ever a minute step), `Chores in town (poverty rebuild after failed retrieval)`. Report only, no gate: bug deaths (autopsy `bug` or beltHP >= 1 with no potion in the last 60 s; needs deaths.jsonl), gold carried at death against gold earned, the per-death table with OPEN / repeat-site / BUG tags.

Fixtures (V, filtered read-only copies of the worker host `soak24/runs/*.log`: only minute, death, state, area_change lines and fights within 10 min after a death): `cmd/d2soak/testdata/soak_a{01,08,11,12,14}_deathrule.jsonl` (a14 is the first 90 min). Test result against docs/plan/death-data.md: a1, a11, a12 FAIL (a1 and a11 and a12 by streak, a12 also stuck at 669 s, a11 also by rate 18.7); a8 WARN only (pair, share 6.6 %, rate, repeat site); a14 no finding. Seconds to field match the oracle table to within 2 s. Not verified: the rule against a live run that has `deaths.jsonl` and the new minute fields (U); `combat fight` lines are stamped at fight end, so the window can end a little late (U, seconds).

## 13. Live alerts from the progress and death rules (SIDECAR; code: cmd/d2soak/live.go, alertcmd.go)

The rules of sections 10 and 12 used to run only in `d2soak report`. They now also run in every alert refresh of a running soak (every 4th sample, about 2 min; the sidecar and `d2soak run` alike) and in `d2soak alerts -in RUN`, so a problem shows in `alerts.json` while it happens instead of in the morning report.

**One judge.** `judgeLog` (live.go) reads the bot log once and runs `AnalyzeMinutes` and `AnalyzeDeaths`; `report` and the live alerts both call it, so they cannot disagree about a finding. The result is cached per log file and limits while the file's size and mtime do not change (a 24 h log is big, the refresh is every 2 min).

**Active findings.** The report lists every finding of the whole run. A finding also carries `Active`: the condition still holds at the end of the log. The live alerts keep only the active ones, so an alert clears when it is resolved:

| code | active while |
|---|---|
| `no_gain` | the stretch since the last clvl, quest-step or gold gain is still at or above the limit |
| `loop_no_gain` | the loop is unbroken at the end (any clvl, quest, gold or 200 xp gain clears it) |
| `idle_in_town` | the newest line is still inside the waiting stretch that reached the limit. A stretch is measured to the line that ends it in the report, so live it trips one minute later than the report's number |
| `state_dominant` | the newest hour block with at least 30 min of data is dominated. Stays through the start of the next hour block until that one has 30 min |
| `below_floor_clvl` | the newest judged checkpoint (hour 2, 4, 8, 12, 24) is still missed by the current clvl |
| `death_streak`, `death_pair` | the death run still ends inside its window (30 or 60 min) of the log end |
| `death_recovery_stuck` | the death is not back in the field, or came back less than 60 min ago; for the share form: the 60 min window ending now is above the limit |
| `death_recovery_share`, `death_rate` | the 60 min window ending now (not the run's peak) is above the limit |
| `death_repeat_site_no_action` | an unbanned repeat-site death happened in the last 60 min |

`alerts.json` is `{updated, level, alerts:[{level, code, msg, since}], status, summary}`. A report `fail` is level `crit` in an alert (the file keeps one vocabulary, `ok|warn|crit`, which `GET /api/v1/soak` already serves); `warn` stays `warn`. `since` is the time the alert first appeared in an unbroken stretch (every alert gets one, the old ones too); it survives a sidecar restart through the existing `alerts.json` and is dropped when the alert clears, so a new raise gets a new `since`. Several findings of one code (idle stretches) are one alert, the newest active one.

`summary` (also the last line of `status.txt`): `live 60m: clvl 12, 8400 xp/h, deaths 1, recovery 3.2%, top state farming 62%`. Window: the last 60 min of minute lines (shorter early in a run); xp/h is `?` before 5 min of data; deaths come from the minute `deaths` counter; recovery is the death rule's 60 min share (`n/a` with the rule off); top state is the busiest `internal/stepgroup` group.

**Limits.** The sidecar does not need a limits block: a `Limits` whose `Progress` or `Death` block is entirely zero gets `DefaultProgressLimits` and `DefaultDeathLimits` (`withProfileDefaults`, the sidecar already starts with profile m24). `runSoak` writes the effective limits to `limits.json` of the run, which is the record `report` later judges with. An explicit block (a rule switched off) stays as given. The old rule stays for `report`/`alerts` on a run directory whose `limits.json` has no `Death` block: nothing new is judged.

**Push hook.** `D2SOAK_ALERT_CMD` (environment of `d2soak sidecar` / `d2soak run`; default unset = nothing is run, nothing contacted). The shell line (`sh -c`) runs on every NEW crit alert (any code: `stall-xp`, `tick-age`, `death_streak`, ...), once per raise: it is not repeated while the alert stays, nor after a sidecar restart (since is read back), but it runs again after the alert cleared and came back. Environment: `D2SOAK_ALERT_LEVEL` (`fail`), `_CODE`, `_MSG`, `_SINCE` (RFC 3339), `_RUN` (run dir), `_STATUS`. 30 s limit, output to stderr, failure only logged. Note for docs/stallfatal.md: d2soak now links `os/exec` for this hook alone, it still sends no game input and acts on nothing else.

Tests (V, offline): `cmd/d2soak/live_test.go`. A line-by-line replay of the M0 fixtures (`soak_a14_minutes.jsonl`, `soak_a{01,08,11,12,14}_deathrule.jsonl`) asserts the alert order (a1: pair warn then streak fail; a12: streak then stuck; a11: streak then rate; a8: never a death fail, share and rate clear again; a14: nothing), that `idle_in_town` raises and clears while `no_gain` stays, and that every fail of the final report was raised live and every live alert is in the report. Writer test: `since` kept, cleared and re-set, hook once per raise, restart does not refire. Not verified: live behaviour on a running soak (U), log growth speed of the judge on a real 24 h log (U; the cache only helps while the log is unchanged, and a growing log is re-read whole every 2 min; the report already does that once).
