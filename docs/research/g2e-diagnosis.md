# Gate 2 g2e: why 0 of 3 seeds passed

> status: current. Read-only analysis of the merged g2e record (docs/soaks/gate2-multiseed-20261106-0225-g2e.md, the three per-seed records, the minute logs and the new `*-g2e-extras.jsonl.gz` logs under docs/probes/soak-logs/), the g2d diagnosis (docs/research/g2d-nokill-diagnosis.md, section 8 for the nokillfix mechanisms) and the bot source of origin/main. No game, no Go changes, no contract change. Tags: V = read from a committed log line (timestamp given, wall clock -05:00; the minute lines are VIRTUAL minutes, so a stretch of minute lines can last 1 s of wall time) or from source; U = inference. Run: 3 seeds (11, 13, 17), all clvl 16, writes after start 0, recovery path A on every seed.

## 1. Findings table

| id | seed | window / time | cause | evidence | proposed fix | est. effect | class |
|---|---|---|---|---|---|---|---|
| E1 | 11 | BELT: minute lines 928-938, 22:22:35-22:23:29 | V: the hero was in a dungeon level (area 11, U: Hole Level 1 by levels.txt order) with 0 potions, life 47%, and NO town portal scroll or tome charge; the walk-to-town fallback does not cover that area; the bot then waited, and the failsafe left the game only after 240 s of no progress | section 2 | F2: in `lowLifeGate` (internal/bot/goal.go) leave the game at once when the heal trip fails with `town.ErrNoPortal` and no potion is left | window 11 -> about 2 minutes | bot defect |
| E1b | 11 | cause behind E1: no scroll while funded (pack 628, stash 2430) | U: why the last town visit (22:20, minute 915-918) left without a scroll is not in the logs | section 2.4 | F2b: scroll reserve as a trip reason and a `tp` field in the minute line | prevents E1 and E2 | bot defect (U) |
| E2 | 13 | NOKILL: minute lines 1015-1026, 21:48:19-21:49:38 | V: out of scrolls and poor (gold 165 then 2), target of the step unreachable by the cave-mouth gate, planner skipping target after target, 2 fresh games, then Recovery | section 3 | F1 (mouth gate) plus F3 (drought timing) | window 12 -> under 8 | bot defect |
| E3 | 13 | deaths 18:29:27 (clvl 4) and 00:59:48 (clvl 16) | V: the first is deterministic (identical to g2d); the second is a 10-archer ranged burst in Tamoe Highland inside the route guard kit rule | section 4 | F6 | small; deaths pass the death rule | bot defect (low) |
| E4 | 17 | fault recovery FAIL | V: a REAL second game cycle 3 s after the recovery (19:24:19), started by the bot, not a counting artefact | section 5.1 | none in the judge; criterion question Q1; bot: do not start a fresh game inside the recovery window | judge could pass it | judge strictness / criterion question |
| E5 | 17 | Andariel FAIL | V: merc died once (18:58) and was never seen again, so the rule needs floor 17 and the hero ends at clvl 16; judge read is correct | section 5.2 | F5 (merc revive never funded, U) | passes if the merc is revived | bot defect (U) plus criterion question Q2 |
| E6 | all | restart causes | V: counts in section 6 | section 6 | F1, F2, F3 | most no-progress minutes | mix |
| E7 | all | nokillfix mechanisms | V: rung3 window fired once, drought stage 1 fired 2/4/2 times, stage 2 can never fire inside the 10 minute limit by construction | section 7 | F3 | see F3 | design flaw |

## 2. Seed 11: BELT window (finding E1)

### 2.1 Where (V)
Replaying the `BeltWindow` rule of `internal/d2dsoakstat/invariants.go` (line 373: a minute line with `HasBelt`, no healing potion in `packHP` or `beltBoxes`, and an area that is not a town) over the minute log gives windows of 8 lines (index 2-9, poverty), and 11 lines, index 928-938, wall 22:22:35 to 22:23:29. Funded: pack gold 628, stash 1802 (the record says stash 2430 at its start of the window; the stash value in the minute lines 928-938 is 1802).

### 2.2 Minute by minute (V, minute lines)
| idx | wall | area | potions | life | state / step |
|---|---|---|---|---|---|
| 925-926 | 22:22:13-35 | 6 (Black Marsh) | 6 | 174, then 113 of 190 | farming, a fight with no kill (kills 6328 twice) |
| 927 | 22:22:50 | 11 | 3 | 86 | "Going to town ([kit: mana potions 0/3 low on mana potions])" |
| 928 | 22:23:04 | 6 | 0 | 128 | farming |
| 929-937 | 22:23:14-23 | 11 | 0 | 91 of 190 (47%) | "Life 47% in area 11, no potion, and the way to town failed 3 times; retrying in 1m53s" |
| 938 | 22:23:29 | 6 | 0 | 190 | farming, first line after the re-entry |
| 939 | 22:23:34 | 1 | 0 | 175 | "Chores in town ([kit: town portal, healing potions" |

Six belt potions were drunk in about two minutes (idx 926-928), then the belt was empty.

### 2.3 Why there was no restock trip (V, extras log, 22:22:51-22:23:24)
- Every town trip fails the same way: `action failed {what: to town, err: town: no town portal available}` at 22:22:51, 22:23:01, 22:23:14, ... and `town trip abandoned` (reasons `[kit: mana potions ...]` and `[0 healing potions low on healing potions]`). The trip TRIGGER did fire (that is the "Going to town" step at idx 927 and the "heal trip" actions); it is the portal that was missing. So it is not the potion policy, not the legacy `t1.town.trigger` path and not a refused trip: the trigger is correct and the means are missing.
- `town.ToTown` (internal/town/trip.go line 56ff) on `ErrNoPortal` calls `walkToTown`, which only works from areas in `townNeighbour` ({2: 1}), `hopToNeighbour` ({8: 2, 3: 2, 17: 3, 9: 3}) or with a usable waypoint (trip.go lines 120-127, V). Area 11 is in neither map, so the fallback returns the bare `ErrNoPortal`.
- `lowLifeGate` (internal/bot/goal.go lines 1457-1495): after `healFailLimit` (3) failed heal trips it calls `lowLifeWait`, a 2 minute hold (`lowLifeRetry`), then clears the failure count and tries again. The log shows 5 `low life and no way to heal: waiting before retrying {retry: 120000000000}` lines (22:23:14.701, :16.917, :19.176, :21.406, :23.711), each followed by the same 4 failed attempts. The waiting is on the game clock and the minute lines run at 1 s of wall time each here, so the 9 minute lines 929-937 are the virtual two minute holds.
- Only the generic failsafe ends it: `failsafe ladder {step 1, what: no progress in state resupplying for 240s}` at 22:23:18.723, step 2 at 22:23:23.126, step 3 `to town` at 22:23:24.191 (`Failsafe: leaving and re-entering the game`), then the re-entry at 22:23:25.869. A fresh game starts in town, so the hero was healed by the leave; the bot then walked into the field (idx 938, area 6, still 0 potions) before the chores ran (idx 939, 22:23:34). That last line is the 11th line of the window.
- No portal was blocked or refused earlier in the minute: `town portal did not appear` and `back to field: portal click failed` lines in this seed are from 21:43 (V) and unrelated.

### 2.4 Why no scroll while funded (U)
The log has no scroll count. Facts that are V: `poverty mode: on` at 21:59:28 (gold 145 below kit cost 190); chore visits at idx 893, 906, 915 (22:17-22:20) with pack gold 281-505 and stash 1802; `town: portal source still empty after restocking, no scroll trip for a while {quiet: 10 min}` at 22:23:35 (after the window, in the first visit after the re-entry). `maxEmptyVisits` (internal/town/chores.go line 664) puts the `tsc` trip reason to sleep for `town.futileMinutes` (10) after two town visits that began with an empty source; U: an earlier pair of such visits, or `town.tpMinGold` poverty logic buying potions first, left the hero without a scroll. Pack plus stash gold is not what the shop spends (the stash is not used at the vendor), so "funded" in the BELT sense (pack + stash >= price) is not "can buy a scroll now" (pack 628 could, so U which rule held it back).

### 2.5 Smallest fixes
- F2 (certain, small): in `lowLifeGate`, when `r.d.Town.ToTown` fails with `errors.Is(err, town.ErrNoPortal)` and `resInput.Belt.HP+RV == 0`, call the same game-leave the failsafe uses (rung 3) at once instead of `lowLifeWait`. New knob `bot.noPortalLeave` (default 1; 0 keeps today's wait). Removes the 9 waiting minutes plus the 240 s ladder lead: the window shrinks to the minutes between the last potion and the leave (about 2).
- F2b (cheap, gives the evidence): add `tp` (scrolls + tome charges) to the minute line (`internal/bot/observe.go` minuteLine, like `fsWhy`); add a trip reason `tsc` when fewer than `town.tpReserve` (new knob, default 2) scrolls are left and gold allows, so a dungeon dive never starts with 0 or 1 scrolls.

## 3. Seed 13: NOKILL 12 minutes (finding E2)

### 3.1 The window is not in Blood Moor and not "Recovery: farming gold" (V)
The 12 lines without a kill or XP gain are minute lines 1015-1026, wall 21:48:19-21:49:38 (kills 6109 and XP 186479 constant from idx 1014 to 1026). The step is `Quest: fighting (Level up to 17)` on lines 1016-1025; the first `Recovery: farming gold` line is 1026, the last of the window. Areas per line: 1, 9, 1, 1, 19, 1, 7, 7, 9, 1, 1, 2 (town and cave hops, not a sweep of one field). `restarts` goes 27 -> 28 (idx 1018) -> 29 (idx 1021). `fsWhy` reads `np_clearing` on lines 1018-1026 ("no progress while clearing"); `navRung` 2-3.

### 3.2 Minute by minute from the extras log (V)
- 21:47:48.630 `poverty mode: on {reason: gold 165 below kit cost 190, no town portal}`; 21:47:48.647 `action failed to town: town: no town portal available`; 21:47:50.196 `nav: escape failed, going on with the ladder {err: town: no town portal available}` after `nav: stalled in a pocket, escaping by town portal` at 21:47:50.192. The hero had no scroll before the window started.
- 21:48:15 idx 1014: trip "Going to town ([kit: town portal, mana potions 1/3 poverty: ...])"; gold 173 -> 102 (a scroll) -> 2 by 21:48:46 (a second 100 gold buy); 21:48:18.909 `town: portal source still empty after restocking, no scroll trip for a while`.
- 21:48:19-21:48:37: the planner skips Catacombs 4, 3, 2 (`no activated waypoint on the way`), Dark Wood and Forgotten Tower with `exit to Cold Plains: nav: click {54 179}: nav: no path: the click would cross the trigger area of an unwanted exit` (the cave-mouth gate, internal/nav/mouthgate.go). Skip lengths 600-1200 s.
- 21:48:37 `xp at risk` + Save and Exit (a save point chicken), 21:48:39 failsafe rung 1 (skip target), a second Save and Exit at 21:48:39.446, 21:48:45 `Cave Level 2` unreachable (same gate error at click {101 84}), 21:48:46 rung 2 and 3 (third leave), 21:48:58 `exit to Outer Cloister: nav: no exit to that level: level 27 from 26` for Jail 1 and Outer Cloister.
- 21:49:16.602 `kill drought {stage 1, area 7, name Tamoe Highland, minutes 8}`, `area skipped {area 20 Forgotten Tower, reason kill-drought}`: the watchdog fired exactly 8 minute lines after the last kill (idx 1022/1023) and skipped the route TARGET (Forgotten Tower), not the transit area the hero stood in.
- 21:49:33-21:49:34 Catacombs 4/3 and Crypt (id 18) unreachable (the same gate error, click {54 179}); 21:49:35.408 failsafe rung 4 (restart the game), 21:49:37 rung 1, `recovery: start (gold 2 < kit 500)`, `poverty mode: on ... no town portal`. Kills resume at idx 1027 in Blood Moor, 11 lines after the last kill.

### 3.3 Should the drought watchdog have broken it earlier, and why not (V code + U)
`droughtStage` (internal/bot/drought.go) arms when `now - max(killXPAt, droughtAt) >= bot.droughtMin` (8) AND the hero stood in a field area for `droughtMinField` (30 s). The clock counts every minute since the last kill, town minutes included, so it fired at the first field step after minute 8: that is on time, it did not "fail to fire". Three weaknesses:
1. The Gate 2 limit is 10; with firing at 8 the stage 1 action has 2 virtual minutes to produce a kill. Here stage 1 (skip the target) re-planned into targets that were all unreachable, so the 2 minutes were not enough.
2. Stage 2 (fresh game) needs another full `droughtMin` after stage 1 (the anchor is `droughtAt`), i.e. 16 minutes without a kill. Under a 10 minute limit it can never rescue a window, by construction.
3. Stage 1 `skipTargetFor` skips the step TARGET, which can be a far area the hero is only passing through. If the planner has no other reachable target, a skip changes nothing.
A goal that names its area (a Recovery or quest-run goal) falls to stage 2, which is the 16 minute path. U: whether the recovery goal can skip at all was not exercised in this window (idx 1026 is its first line).

### 3.4 Root cause and fix (U for the nav part, V for the symptom)
The same `exit to Cold Plains ... unwanted exit` failure appears 104 (s11), 295 (s13) and 47 (s17) times; seed 13 repeats click `{54 179}` 143 times and `{113 168}` 93 times, seed 11 click `{148 337}` 117 times (V, extras). The click is the exit click of the leveling target itself, and it is refused because the straight segment comes within `mouthTriggerR` (15 subtiles, a constant) of an unwanted exit; the gate holds after `gateMaxRewrites` (3) rewrites and the walk fails, so the whole target is skipped for 150-1200 s. U: which unwanted exit is near; the error does not name it.
- F1: make the gate a knob and give the wanted-exit approach a way through: in `internal/nav/mouthgate.go` `gateClick`, when the click target is within `mouthTriggerR` of the exit the step WANTS (`n.wantExit`) and also within it of an unwanted one, approach from the side away from the unwanted exit (waypoints at 20+ subtiles, as the file comment says long step-aways never fired) instead of failing; add knobs `nav.mouthTriggerR` (15) and `nav.gateMaxRewrites` (3); log the unwanted exit position and distance in `ErrMouthGate`. Effect: removes most of the "unreachable target" no-progress minutes (section 6, rank 1).
- F3: drought timing (see section 7).

## 4. Seed 13 deaths (finding E3)

### 4.1 18:29:27, Gargantuan Beast, Cold Plains, clvl 4: same cause as g2d (V)
`character died {area 3, killer Gargantuan Beast, x 174, y 316, corpseValue 168, maxLife 69, heal 0, beltMP 2, tome 2}` in g2e (18:29:27.010) and, byte for byte in those fields, in g2d (2026-11-05 08:10:49, minute index 67 both times). With lockstep and the same seed the game replays; the two deaths are the same death. Context (g2e minute lines 60-67): step `Quest: fighting (Level up to 5)` in Cold Plains at clvl 4 with 69 maximum life; kills 218 -> 304 in about 30 s of wall; `dmgTaken` 332 -> 650; potions 6 -> 4 -> 1 -> 5 -> 0; 12 hostiles near; autopsy `5 potions drunk in the last minute`, melee, distance 2. The chicken line is 25% (`combat.chickenPct`): 69 x 0.25 = 17 life is smaller than one beast hit, so the chicken cannot save it (U on the hit size). The level plan sends a 4th level 69-life caster into a pack area; the autopsy's monster level `mlvl=2` is 2 below the character, so the death diff rule sees nothing wrong (U: the Cold Plains level in the table is the area base, not the beast).
Smallest fix (U on effect): raise the first level a Sorceress may use Cold Plains for the "Level up to N" step in the plan table (internal/plandata, a data change, no code) or keep her in Blood Moor until clvl 6; cheaper still `combat.crowdN` (8) lowered for clvl under 6 (new profile field). The death rule is not violated (2 deaths in 12945 kills, the record says PASS), so this is low priority.

### 4.2 00:59:48, Returned Archer, Tamoe Highland, clvl 16 (V unless marked)
- Tamoe Highland (7) is under the route guard (internal/bot/routeguard.go; `route.monasteryMinLevel` 14, kit: 4 healing, 2 portals, mana for casters). The guard was OPEN and the hero was allowed in: clvl 16 >= 14, belt 6 healing and 2 mana potions on idx 1892-1895 and again 1897 (belt mana count 2 each, from `beltBoxes`). The hero had been in 7 and in Monastery Gate (26) on idx 1893-1895 (00:57:57-00:58:56) trying `exit to Outer Cloister` for Jail 1 (`no exit to that level: level 27 from 26`, 00:58:13).
- 00:59:33.489 `xp at risk {why: save point}` + Save and Exit at life 143 of 201 (71%); re-entered 00:59:35.144; idx 1896 town chores "heal low on mana potions"; idx 1897 (00:59:43) area 7, life 193/201, kills +10; idx 1898 (00:59:49) dead. `dmgTaken` 23332 -> 23525 = 193 damage in that one virtual minute: the hero lost all its life to 10 ranged hostiles in one minute line.
- Autopsy: `fighting in Tamoe Highland, killed by Returned Archer; 10 hostiles near; mana 32%; belt 5 hp 1 mp, 1 portal scrolls, 3 potions drunk in the last minute`, `lastSafety: drink-life`. So the drink rule fired (3 Minor Healing potions) and the chicken (25%, about 50 life) did not get a chance: 193 life went in less than one decision gap. The retreat ladder is off (legacy path, per the brief), and `combat.crowdN` (8) counts hostiles within 8 subtiles; archers shoot from farther away (U).
- Why the hero went there: it did not decide to; the guard passes and the step `Level up to 17` routes through Tamoe to the Monastery road (V: idx 1893-1895). The retry after the death shows the guard working with a lower kit: 01:00:13 `Tamoe Highland (7): route guard (0 mana potions < 2 (caster))`.
- Smallest fix: potions of the Minor class (45 life) at 201 maximum life are a thin margin; the belt in all three seeds is Minor Healing at clvl 16 (V: idx 1892). Cheapest: stop re-entering the dangerous road right after a save-point chicken: re-check the route guard with the stricter "after a death or chicken in the last 10 minutes" kit (4 healing, 2 mana, 2 scrolls) in `areaGuardWhy`; knob `route.recentChickenKit` (new, default on). Better but bigger: buy Light/Regular healing potions once gold allows (town shop, out of scope of this note). U: effect; the death rule passes, so this is not a gate failure.

## 5. Seed 17 judge results

### 5.1 Fault recovery FAIL: a real second restart, not a judge artefact (E4)
- The judge (ops repo `tools/gate-judge.py`, fault section, "ra - rb == 1") compares the `restarts` field of the last minute line before the kill with the first after it: seed 17 shows `restarts 2` on idx 408 (19:24:14) and `4` on idx 409 (19:24:22).
- V, extras log: kill at 19:24:15.9 (`d2d process exited {state: signal: killed}` 19:24:16.669, `game lost, restarting it`, re-entered 19:24:18.091, `xp lost 1372`: saved XP 83703 equals the record's XP after). That is restart one.
- V: at 19:24:19.163-19:24:19.167 the planner rejects three leveling targets ("area not allowed now: Tamoe Highland (7): route guard (no merc, clvl 11 < 14)", Forgotten Tower (clvl 11 < 15), Outer Cloister) and at 19:24:19.168 the bot requests Save and Exit; `d2d: leave: re-entered the game` at 19:24:20.807. That is restart two, started by the bot 1 s after the first recovery. No failsafe ladder line sits between (V), so the path is the leveling "no target, fresh game" decision (`noTarget` / `freshGameToLevel`, internal/bot/task.go lines 949-970 and goal.go 1005-1020; U: the INFO line that would name it is not in the extras log).
- Seeds 11 and 13 had no such second cycle (+1 each), so seed 17 lost on a coincidence of the fault with a target-less planner moment. It is a bot defect (a fresh game right after a recovery wastes the recovery window and double counts) and a criterion fragility, not a judge bug: the contract says "restarts rises by exactly 1" (docs/contracts/soak-gates.md, Injected fault recovery) and "Double restart ... FAIL", and the judge implements it literally.
- A judge fix would only be a BUG fix if the contract meant "restarts caused by the kill". The contract wording does not say so; see question Q1. Bot fix: do not start `freshGameToLevel` within 180 s of a lost-game recovery (new knob `bot.freshAfterRecoverySec`, default 180); idle in town instead.

### 5.2 Andariel FAIL: a real behaviour; the judge read is correct (E5)
- The interim rule in the judge (`andariel_verdict`) and the contract (Held with merc alive / Held with merc dead scenarios): a `boss: andariel gate pass` or `boss: held for andariel (` line with the latest `route guard: andariel floor=F (merc=S)` before it, at clvl >= max(14, F-1) with the merc alive, or clvl >= F (17) with the merc dead or none. It reads the merc state from the floor line itself, not from `merc: seen alive`.
- V: seeds 11 and 13 produced 6 and 11 gate/held lines at an adequate clvl (floor 15, merc alive). Seed 17 produced one held line at clvl 9 (floor 17, merc none) and no other. The evidence line "merc read alive once" is the count of `merc: seen alive=true` lines: 1 in seed 17, 47 in seed 11, 45 in seed 13.
- V (minute and extras logs): seed 17's merc died once, 18:58; no `merc: died` or `seen alive` line follows, so the merc was never revived (gold peaked at 1245 and `poverty mode: on` fired 57 times). All three seeds end at clvl 16 in area 6, `Quest: fighting (Level up to 17)`: clvl 14 at 20:52, clvl 15 at 22:14, clvl 16 at 00:14 (seed 17); seed 11 and 13 reached 16 at 00:00 and 23:25. With a dead merc the floor is 17 and 27 virtual hours do not get there.
- So not a misread merc state (the merc was dead or gone) and not "the hero never reached Andariel" as the sole reason: with an alive merc seeds 11 and 13 held at the gate at clvl 14-16; seed 17 would have needed the merc.
- Cause (U): no revive. `merc.reviveGoldMult` (1) plus `merc.kitReservePct` (100% of a full kit refill) means revive needs price + full kit in pack gold; the 670 recovery minutes (37% of the run) show the character was too poor to meet it. The log does not carry the revive decisions.
- Fix F5: log the revive decision (`merc: revive blocked {price, gold, kitReserve}`) at INFO once per 10 minutes, then lower `merc.kitReservePct` (100) to 50 for the first revive of a merc that has never been revived, or fund the revive before the second potion refill. Needs the log line first (next run).

## 6. Dominant restart causes and minutes lost (E6)

Counts from the extras logs (V): Save and Exit 194 / 208 / 138 (s11 / s13 / s17), `failsafe ladder` 62 / 89 / 17, `back to field: portal click failed` 33 / 35 / 39, `nav stuck` 2563 / 3957 / 3154, `xp at risk` 61 / 87 / 44.

Healthy (survival or by design): the `xp at risk {why: save point}` leaves (61 / 87 / 44) are scheduled save points, and bursts of 6-7 Save and Exit lines 2-3 s apart (15 / 8 / 9 bursts, 86 / 47 / 59 of the leaves) match the recovery farm cycling new games (U: the cause line is not in the log; the bursts sit in `Recovery: farming gold` time). They cost no kill minutes by themselves.

Minute lines with no kill or XP gain whose wall interval contains an event of the class (V, scripted over the minute and extras logs; classes overlap; 312 / 542 / 353 zero-progress lines in total, about 65% of them contain no logged event, i.e. ordinary walking and clearing):

| rank | class | s11 | s13 | s17 | what it is |
|---|---|---|---|---|---|
| 1 | unreachable target (`leveling area unreachable`, `leveling target not reachable`) | 43 | 121 | 25 | the cave-mouth gate (`exit to ... unwanted exit`, 104 / 295 / 47) and `no exit to that level` (14 / 60 / 18) turn a reachable target into a 150-1200 s skip (F1) |
| 2 | game leaves (Save and Exit, healthy ones included) | 45 | 72 | 32 | about half chain behind a failsafe or a stuck walk (F1, F2) |
| 3 | failsafe ladder (steps 1 to 4, `no progress in state ...`) | 35 | 52 | 7 | consequence of 1 and of `town: no town portal available` (action failed 150 / 101 / 83, section 2) |
| 4 | no portal / trip abandoned | 20 | 9 | 5 | the E1 chain; high leverage because it ends in the BELT failure |
| - | `back to field: portal click failed` | 14 | 16 | 14 | a recovered nuisance: 16-19 `portal ... dropped` and `walk` fallbacks per seed, no window |

Top 3 defects by minutes lost: (1) the mouth-gate/no-exit unreachable targets (F1), (2) leaves chained behind failsafe ladders and stuck walks (consequence of 1 and 3), (3) no town portal in the field (F2). `nav: no path ... trigger area of an unwanted exit` is the single biggest named defect; `back to field: portal click failed` is not among the top three.

## 7. Did the nokillfix mechanisms fire as designed (E7)
- rung3 window (`bot.rung3WindowMax` 3 per 20 min): fired once, seed 11 at 22:14:26 (`failsafe: 3 rung 3 cycles within 20 min -> field reset`, V). Seeds 13 and 17 had none. Fired and did what was designed (no new game, skip the area). The seed 11 g2d cycling mode did not recur.
- Drought stage 1: 2 / 4 / 2 (seed 11 at 22:52:32 and 22:54:16, seed 13 at 21:08, 21:49, 22:24, 23:40, seed 17 at 21:34 and 23:34, V). Stage 2 never fired. By construction (section 3.3): stage 2 needs a further `bot.droughtMin` (8) from stage 1, i.e. 16 minutes, while the criterion tolerates 10; and a kill after stage 1 resets the series. So stage 2 is a dead stage for the gate.
- Empty-field advance (`bot.emptyFieldSec` 150): it IS logged. It shares the message of the general empty-area leave: `area abandoned: no kill for a while {area, name, seconds}` (internal/bot/goal.go line 1379). Seen 1 / 4 / 5 times (seed 17 in areas 4, 4, 5, 6, 6, V). Nothing in the line says which limit tripped (U), so add `limit` (150 vs `EmptyAreaSec`) to it.
- Net: the g2d empty field window of seed 17 did not recur (longest window 10 lines, passing); the seed 13 window of 12 lines is a cousin of the g2d seed 11 pattern but with a different root (E2).

## 8. Ranked fixes
1. F1 nav mouth gate (internal/nav/mouthgate.go `gateClick`; knobs `nav.mouthTriggerR` 15, `nav.gateMaxRewrites` 3; log the unwanted exit): largest class of lost minutes; removes the E2 root. Effort medium; needs a drill at a Blood Moor exit (offline mem.Fake map test; two maps).
2. F2 low-life/no-portal leave (internal/bot/goal.go `lowLifeGate`, knob `bot.noPortalLeave`) plus F2b scroll reserve (internal/town/chores.go `tripReasons`, knob `town.tpReserve`, minute-line field `tp`): fixes E1 directly, one small function. Effort small.
3. F3 drought timing (internal/bot/drought.go `droughtStage`): lower `bot.droughtMin` 8 -> 5, add `bot.droughtStage2Min` (3, counted from stage 1) so a fresh game can happen inside 10 minutes; skip the CURRENT area (or the hero's current route hop) as well as the target. Effort small; risk: more skips in slow fields (Gate 1 balance).
4. F4 post-recovery quiet: no `freshGameToLevel` within 180 s of a lost-game recovery (`bot.freshAfterRecoverySec`). Effort small; makes the fault criterion robust.
5. F5 merc revive visibility and funding (log first, then `merc.kitReservePct`). Effort small plus one run.
6. F6 deaths: route guard "recent chicken" kit; Cold Plains entry level in the plan table. Effort small, low value for the gate.
7. F7 observability for the next run: `tp` in the minute line, `limit` in the empty-area line, the revive decision, the chicken reason on every Save and Exit.

## 9. Criterion questions for the USER (max 4)
Criteria and Numbers tables are not touched by this note; these are questions only.
- Q1 Fault recovery: should "restarts rises by exactly 1" mean "the kill caused exactly one `game lost, restarting it`", ignoring unrelated bot-initiated game cycles in the 3 minute window? Today seed 17 fails because the bot itself started a second game 1 s after the recovery. Recommendation: keep the criterion literal for now and fix the bot (F4); if the user wants it kill-caused only, the judge needs a change (count `game lost, restarting it` events plus a separate INFO line for other cycles).
- Q2 Andariel interim rule: with a dead or never-revived merc the floor is 17 and a 27 virtual hour run ends at clvl 16, so the criterion is unreachable without a revive. Recommendation: keep the rule (it tests that the merc policy works) and treat a missing revive as the bot defect F5; do not lower the floor.
- Q3 NOKILL margin: the limit is 10 virtual minutes and the drought watchdog fires at 8; any single hard window decides the gate (g2d note section 6). Recommendation: keep 10 and fix the bot (F1, F3); do not widen the limit until F1 and F3 are measured.
- Q4 BELT funded windows: today the rule counts pack plus stash gold as "funded", but the vendor only spends pack gold. Recommendation: keep it as is (it is the stricter reading and E1 is a real defect: 628 pack gold); no change.

## 10. What stays unknown (U), to capture next run
- Why the hero left town without a scroll (E1b): add `tp` to the minute line and keep INFO lines `town: bought`, `town: portal source`.
- Which unwanted exit the gate sees at the repeated clicks (F1): log position and distance.
- The cause lines of the 6-7 Save and Exit bursts and of the post-recovery fresh game (E4).
- The merc revive decisions of seed 17 (E5).
- Method note (V): all counts in this note come from scripts over the committed `*-g2e-minutes.jsonl.gz` and `*-g2e-extras.jsonl.gz`; the extras logs hold WARN and a few INFO lines, so absence of a line proves nothing about INFO-level events.

## 11. F2-F5 fixes (branch agent/g2efix, offline tested, no live proof)

Tags: V = read from code or a test run, U = not proven live. The next Gate 2 is the live proof.

- F2a no-portal low life (V cause: section 2.3). `lowLifeGate` now remembers the first heal trip that failed with `town.ErrNoPortal`; once `healFailLimit` rounds failed and `bot.noPortalWaitSec` (20 s, 0..240) passed it starts a fresh game (`noPortalLeave`, internal/bot/goal.go) instead of `lowLifeWait`. A new game starts in town, which heals. 240 keeps the old two minute waits. Cycler missing: the old wait.
- F2b scroll reserve (U cause: the log has no scroll count; the code path found is the poor cap in `internal/town/shop.go` that held the scroll target at 1 under the trip gold floor, so a funded-looking hero left with one scroll). New knob `town.tpReserve` (2, 1..5): a poor hero that already carries its minimal healing kit (`resource.kitMinHP`) buys that many scrolls; with no potion carried the potions still come first (1 scroll). 1 is the old cap. The `t1.tp.lost` ladder point is untouched (the legacy test BuyKit is unchanged).
- F2c walk-back table (S: nav `act1Hints`, U: never walked live). `resource.chainHome` gains 11, 13, 14, 15, 18, 19, 20 (holes, second caves, crypt, mausoleum, Forgotten Tower, passage 2). Tamoe Highland (7) and the pits stay out (codefix F1 test).
- F3 drought timing (V code). New `bot.droughtStage2Min` (2, 0..30): stage 2 (fresh game) follows stage 1 after 2 virtual minutes from the stage 1 action, so 8 + 2 stays inside the 10 minute limit. 0 restores a full `bot.droughtMin`. A goal that cannot skip an area already took the fresh-game path at stage 1 (`skipTargetFor` returns false for non-leveling, non-camp goals); unchanged.
- F4 quiet after recovery (V cause: section 5.1). New `bot.freshAfterRecoverySec` (180, 0..900): `freshHold` forbids the planner's fresh game (`noTarget`, `freshGameToLevel`) and the drought fresh game for that long after a lost-game recovery (`loop.gameLost`), unless life is below the resume line. The judge and criteria are untouched.
- F5 merc revive (U cause: gold). V: `ReviveStep` ran AFTER `Townsman.Chores`, so every purchase (and the bank deposit) came first. `mercTown.Chores` now calls `ReviveFirst` before the shop: pack gold only, kit test waived because the kit reserve (`merc.kitReservePct`) is inside the bar. The normal step after the chores stays. The refusal line (`merc: revive skipped (reason) price= gold= need=`) is now WARN, once per town visit (2 minute gap), so the extras log carries it.

Unproven live: all five. Ladder parity tests (internal/ladder, internal/decide) pass unchanged.
