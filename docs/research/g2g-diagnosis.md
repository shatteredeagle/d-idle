# Gate 2 g2g wave 1: diagnosis of the failures (seeds 11, 13, 17)

> status: current diagnosis, 2026-11-07. Task TASK-65 (record) and TASK-66 to TASK-68 (proposed fixes). Docs only: nothing was run live, no contract or Numbers line is edited here.

Tags: V = read from the committed minute and extras logs (`docs/probes/soak-logs/*g2g*`) or from the full bot logs of the run on the worker host (read-only; line = log time, wall clock, EST) or from code at bot commit 2a6cf2b; U = inference, not proven. "Line N" is the index of the `minute` line (the judge's minute number is the same index, off by one in window length). The record is `docs/soaks/gate2-multiseed-20261107-0150-g2g.md`. Wave 2 (seeds 19, 21) was not touched.

## 1. Findings table

| id | seed | window | cause | evidence (log time, line) | proposed fix | est. effect | class |
|---|---|---|---|---|---|---|---|
| F1 | 13 | NOKILL 11 (lines 1189 to 1201, 20:15:46 to 20:16:49) | V: equip failure leaves the game, then a nav mouth-gate refusal walks the failsafe ladder to a game restart | 20:15:28 and 20:15:41 `leveler: equip failed` (Cap 12.0 to head, slot click refused, still on cursor), each followed by `d2d: Save and Exit` and a re-entry in 1 to 4 s; 20:16:13 ladder step 1 `exit to Cold Plains: nav: click {49 179}: nav: no path: the click would cross the trigger area of an unwanted exit`; 20:16:14 to :16 ladder steps 2, 3 (leave) and 4 (restart game); 20:16:26 kill drought stage 1 (10 min); kills frozen at 7830 from line 1190 to 1201; restarts counter 42 to 44 inside | N1 (mouth gate loop) and E1 (equip failure leaves the game); see 3 | removes the 2 to 3 min of ladder (about 3 of the 12) | bot defect |
| F2 | 17 | NOKILL 14 (lines 1035 to 1050, 20:40:28 to 20:41:51) | V: no town portal scroll or tome, hero in a nav pocket in Black Marsh, walk to town also fails; only game restarts free it | 20:36:40 `safety action failed ... combat: no town portal scroll or tome`; 20:39:48 `exit to Dark Wood ... unwanted exit`; 20:40:21 `town trip abandoned [no town portal scroll]`; 20:40:38, :41, :44, :55, 20:41:04, :11, :21 `nav: stalled in a pocket, escaping by town portal` at {87,287}; 20:40:44, :54, :58 `escape failed ... town: no town portal available (walk to town failed: nav: stuck at {87 287} ... / no exit to that level: level 5 from 6)`; drought stages 1 (8 min) 20:41:07, 2 (10) :22, 3 (13) :41; ladder rungs 3 and 4 at 20:42:01 and :04; restarts 10 to 13 inside | S1 (scroll first), N1 (the pocket escape), and the ladder is slow (see F5) | would end the window at about 8 min | bot defect |
| F3 | 17 | BELT 8 (lines 198 to 205, 17:56:30 to 17:57:56) | V: NOT the injected fault. Two merc revives at clvl 8 emptied the purse; the judge counts the window as funded only because pack gold touched 31 on its last minute | the fault is at 18:29:51/52 (line 385, `d2d process exited`); this window is 33 min of wall time earlier. 17:53:51, 17:55:44, 17:56:47 `merc: died` x3; 17:54:53 and 17:56:01 `merc: withdrew 260/250 gold for the revive` + `merc: revived price=270`: stash 586 to 326 to 0 (lines 194, 198); pack gold 16 to 20 on lines 198 to 205 (potion price 30); line 206 gold 31, trip to town, potion bought | policy consequence of "revive before potions" (user 2026-11-06) at a poverty point; judge edge: see 4 | one window per seed at most | judge + criterion question |
| F4 | 11,13,17 | merc | V: the merc is dead 83 to 96 percent of the time; see 2 | | M1 to M3 | the largest lever on gold-farm time and Andariel | bot defect |
| F5 | all | failsafe | see 5 | | | | mixed |
| F6 | all | early-death rules | V: they fired; the extras filter hid them; see 6 | | filter fix (ops tool) | diagnosis only | tooling |
| F7 | all | mouthRef | V: refusals remain, the detour almost never fires; see 7 | | N1 | | bot defect |
| F8 | 13,17 | windows with restarts | see 8 | | | | criterion question |

## 2. The mercenary (item 4)

### 2.1 Revives, cost and alive time (V, full bot logs)

| per seed | 11 | 13 | 17 |
|---|---|---|---|
| `merc: died` | 17 | 17 | 16 |
| `merc: revived` (success lines) | 16 | 16 | 15 |
| revives per wall clock hour (hours 0 to 7) | 2,2,2,2,2,2,2,2 | 2,2,2,2,2,2,2,2 | 2,2,2,2,2,2,2,1 |
| gold paid (sum of the read quotes in `revived price=`) | 3,030 | 3,030 | 4,500 |
| per wall hour / per virtual hour | 379 / 93 | 379 / 75 | 563 / 159 |
| gold picked up (sum `goldPickup`) per wall hour / per virtual hour | 3,049 / 745 | 3,366 / 669 | 2,435 / 688 |
| revive share of the pickup | 12 percent | 11 percent | 23 percent |
| `merc: seen alive=true` of all `seen` reads (1 per wall minute) | 53 of 461 | 19 of 457 | 76 of 455 |
| time-weighted alive fraction | 11.5 percent | 4.2 percent | 16.8 percent |
| merc state at the end (last `seen`, Andariel hold) | dead (level 16) | dead (17) | dead (16) |
| `revive skipped` reasons | cap 52, gold 1 | cap 52 | cap 58, gold 3 |
| time in `gold farm` mode (wall min of 480) | 296 | 165 | 272 |
| wall minutes alive after each revive (16/16/15 samples) | 0.1 to 11.5, median about 1.5 | 0.2 to 3.0, median about 0.9 | 0.2 to 15.2, median about 2.3 |

The revive success line `merc: revived` and `merc: withdrew N gold for the revive` are INFO lines: they are in the full bot logs, not in the extras logs (see 6). The state `d2d` is not needed: the `seen`, `died`, `revived` lines are enough.

Findings:
- M1 (V, code). The revive cap is counted on the WALL clock. `cmd/d2bot/mercwire.go:210` builds the wire with `now: time.Now`; the rest of the d2d backend uses the env clock (`cmd/d2bot/d2dbackend.go:183`, virtual under lockstep). Gate runs use `--lockstep` and run 4.1, 5.0 and 3.5 virtual minutes per wall minute (1963/2414/1699 minute lines in 480 wall min). So `merc.reviveMaxPerHour` (2) and `merc.stopDeathsPerHour` (3) mean 2 revives per roughly 4 game hours in a soak. The revive log confirms it: exactly 2 revives in every wall hour, back to back at the start of the hour (for example seed 11: 19:47:21 and 19:48:49), then dead until the next clock hour. In a real-time game the design (2 per game hour) holds, so this is a lab artifact on top of a real problem (M3).
- M2 (V). The price used for the cap lines, `need=` and the gold-farm trigger is the formula estimate (`ReviveCost`, level squared / 2 x 15: 1,470 at clvl 14, 1,920 at 16, 2,160 at 17), but the quote read from Kashya is 60 to 270 (`merc: revive quote read price=120 gold=1224`, seed 11 19:47:21; `price=180` at clvl 15, seed 13 20:54:14; `price=270` at clvl 8 and 12, seed 17). The contract says the estimate MUST NOT be treated as the price (`docs/contracts/merc-policy.md`, "Revive price is read, not computed"), but `gold farm on need=1730 gold=878 clvl=15` (seed 13, 20:03:14) shows the farm trigger using it. Effect: the character is confined to the gold-farm zone list (Blood Moor ... Crypt) for 296, 165 and 272 of 480 wall minutes (62, 34, 57 percent) while the merc is dead and gold is under about 1,500 to 2,200, to afford a revive that costs 120 to 270. XP effect on the clvl 13+ bands: U (the per-band XP/h at clvl 13+ is the slow part, 5,720 to 7,653).
- M2b (V, minor). `merc: withdrew 840 gold for the revive` (seed 11 19:48:49) for a price of 120: the stash withdrawal is sized from the estimate and moves gold into the pack; no gold is lost, but the pack then reads under the farm need and turns the farm mode on again at 19:49:07 (`need=1520 gold=840`).
- M3 (V for the numbers, U for the cause). The merc dies within about 0.1 to 15 wall minutes of a revive (median 1 to 2.3, so 4 to 10 game minutes). Deaths per ALIVE hour are therefore about 15 to 50, not 2: the "about 2 deaths an hour" is the revive cap, not the merc. Examples: seed 11 revived 19:47:21, died 19:47:50, revived again 19:48:49, died 19:49:00 in Tamoe Highland with `merc: down in a crowd, leaving action=crowd-retreat crowd=8` at 19:47:15 and 19:49:43 and the hero drinking healing/rejuvenation every few seconds (hero life 43 to 64 percent). The merc is a Rogue Scout (class 271) of level 7 to 17; table HP is about 45 at level 3 growing 8 per level, so roughly 100 to 170 HP (`docs/sme/mercenary.md` section 4, V table). Deaths by area at the time of the `died` line: seed 17 Black Marsh 9, Tamoe Highland 3, Burial Grounds 3, town 1; seed 13 Black Marsh 5, Dark Wood 3, Hole Level 2 2, town 2; seed 11 Stony Field 4, town 4, Cave 1 2, Black Marsh 2. Share of the minutes at clvl 12+: Black Marsh 18 to 25 percent, Stony Field 18 to 22, Dark Wood 14 to 16. So no strong area clustering beyond time spent; Black Marsh is over-represented in seed 17 only (9 of 16 deaths against 25 percent of the time). 7 of the 50 deaths are logged while the hero stands in town (`Returning to the field`, `Chores in town`): a merc cannot die in town, so these are deaths noticed on return through the portal (the merc was left in the field) or a state-read lag (U). Deaths do not correlate with clvl in a way these 50 samples can show (U).
- M4 (V, arithmetic). The stop rule never fires. `ShouldRevive` raises the bar to 5 x price only when `DeathsLastHour > stopDeathsPerHour` (3, `internal/merc/policy.go:231`). With the cap at 2 revives an hour there can be at most 2 revives plus the death that preceded them, so at most 3 deaths in a clock hour: the condition is never true. No skip reason `stopped by deaths` appears in any log (reasons are `revive cap` and `gold` only). The cap does what the stop rule was meant to do (it stops reviving), for a different reason. SME's "3 or more deaths an hour means the zone is too hard" signal exists in the data (the hero drinks heavily in the same crowds) but nothing consumes it to change the AREA; that would be a new rule, not a fix of this one.
- Andariel interim rule (V). All three seeds ended with the merc dead (`merc: seen alive=false`: seed 11 01:27:37, seed 13 01:28:51, seed 17 01:29:01), so the floor was 17 (`route guard: andariel floor=17 (merc=dead)`, seeds 11/13/17 at 01:23:28, 01:02:46, 01:18:38) and clvl was 16, 17, 16. Seed 13 reached 17 at 01:02 with a dead merc; the boss run was still not started because of `boss: held for andariel (unattended hold: the throwaway fight has not won yet (boss.andarielHold))`. So the interim rule is satisfied by the hold line, not by a live merc. With the merc alive 4 to 17 percent of the time, "live merc at clvl 15" will almost never be true when the hold lifts: the revive cap and the short life together make the 15 floor unreachable in practice (U for the lift moment).

## 3. Mouth gate and equip (F1, F2, item 7)

N1 (V). The mouth gate (#254, `internal/nav/mouthgate.go`) is in the bot (commit c13d9ba6 is an ancestor of both 65a7013 (g2f) and 2a6cf2b (g2g)). Counts from the full logs:

| per seed | 11 | 13 | 17 |
|---|---|---|---|
| `click rewritten` how=away / clip / detour | 477 / 152 / 10 | 197 / 545 / 17 | 528 / 123 / 2 |
| `click held` why=no safe click / rewrite cap | 333 / 46 | 462 / 6 | 149 / 38 |
| `action failed ... unwanted exit` in the extras (task step / recovery / guard exit) | 147 / 67 / 23 | 263 / 63 / 1 | 53 / 10 / 0 |
| `mouthRef` summed on minute lines (g2f 430/536/159) | 422 | 468 | 206 |

The detour fires 2 to 17 times per seed against 187 to 468 holds. The holds repeat at a few spots: seed 11 from {144,342} to {148,337} (38 times in the encampment area label alone, 22 more under another), seed 13 {50,180} to {54,179} (56 + 50) and {114,169} to {113,168} (54 + 44), seed 17 {114,169} to {113,168} (16 + 7 + 26). The same short step (4 to 5 subtiles) is refused again and again; this is a loop, not a one-off. Cause inside the function: U. The refusal comes from `gateRewrite` returning `how == ""`, i.e. no candidate passes `clickOKAll`; for a hero already inside the trigger disc (`d0 < mouthTriggerR`) the "away" branch tries 3 lengths x 7 angles on walkable cells and the detour is only tried from outside the disc, so inside-the-disc holds are never detoured. The area labels above use the nearest minute line and are approximate (areas change every few seconds around restarts). Verdict on #254: it removed no refusals (g2f and g2g sums are within noise, 422/468/206 vs 430/536/159); where it works it is the `away` and `clip` rewrites (1,100+ per seed), which are not counted in `mouthRef`.

E1 (V for the sequence, U that it is causal). `leveler: equip failed` 7, 20, 9 times per seed (`slot click refused ... stillOnCursor: true`; seed 13 includes Cap, Leather Gloves, Boots swaps). In seed 13 at 20:15:28 the failure is followed in the same second by `d2d: Save and Exit requested (chicken / leave game)` and a re-entry at 20:15:30; the same at 20:15:41. The extras show 157/236/92 `Save and Exit` lines, of which `xp at risk why=save point` is 48/99/40. A failed equip that leaves the game costs a fresh game each time. Whether the leave is the equip handler or the generic save point is U (the cause chain needs the debug lines).

## 4. The judge and the criterion (items 3, 8)

Counted on the minute logs and the windows in `d2dsoak` output of the record:
- Over-limit window lines: 3 (seed 13 NOKILL, seed 17 NOKILL, seed 17 BELT). All 3 contain at least one fresh-game restart (restarts counter: 42 to 44; 10 to 13; 2 to 3 inside the window). Windows at 9 to 11 lines that are not failing also contain restarts in some cases (seed 17 lines 399 to 408: 4 to 5). Zero failing windows are restart-free.
- The criterion text says nothing about restarts: `docs/contracts/soak-gates.md` line 160, "no state may last more than 10 virtual minutes without a kill or XP gain"; line 204, "no funded window over 5 virtual minutes with zero healing potions outside town ... A window is funded when `gold` + `stashGold` reaches one Minor Healing Potion (30 gold) at some minute; others are poverty, not failing".
- A restart is the failsafe's REMEDY for a stall (ladder steps 3 and 4), so it sits inside almost every long stall by construction; excluding windows that contain one would exempt exactly the stalls the criterion exists to catch. Recommendation: keep literal (user question 1).
- BELT judge edge (F3): `BeltWindow.Funded()` is `TotMax >= PotionPrice` over the whole window (`internal/d2dsoakstat/invariants.go:162`), so a window that starts in poverty is counted as funded if pack gold reaches 30 on its last minute (here 31 on the minute the hero walked to town). Seed 17 line 198 to 205: gold 16 and stash 0 for 7 minutes. Option for the user: judge funded at the window's FIRST minute (money available while out of potions), which would classify F3 as poverty. This changes a judge rule, not a Numbers table value; it still needs the user.

## 5. Failsafe reasons and fresh games (item 5)

Minute-line counts (V; `fsWhy` marks the last failsafe reason on a minute line, not minutes lost):

| reason | seed 11 | seed 13 | seed 17 | what it is |
|---|---|---|---|---|
| np_clearing (`no progress in state clearing for 60s`) | 15 | 50 | 11 | the stall detector; fed by N1 and by pockets |
| task_step | 19 | 20 | 8 | a quest or travel step failed: mostly `exit to X ... unwanted exit` and `no exit to that level` |
| recovery | 6 | 11 | 4 | poverty restock cycles (`recovery: start (gold N < kit 500)`) |
| guard_exit | 4 | 1 | 2 | potion or crowd guard leaving an area |
| fresh games (minute `restarts`) | 41 | 88 | 25 | 157/236/92 `Save and Exit` lines |
| zero-kill minute lines | 364 (18.5%) | 504 (20.9%) | 378 (22.2%) | |

Ranking by likely minutes lost (U on the exact minutes; the kill-less minute share is V):
1. Nav stalls from mouth-gate refusals and pockets (np_clearing + task_step): 34/70/19 minute lines and 3,307/4,091/3,201 `nav stuck` lines, 7/6/11 pocket escapes, 25/13/26 `town portal did not appear`. Defect (N1, S1).
2. The merc cycle: gold-farm confinement plus potion drain at revive points (M2, F3). 34 to 62 percent of the run under farm restrictions. Defect (M1, M2).
3. Equip failures and the leave-game per failure (E1) and the save-point leaves (48/99/40): healthy in intent (they protect XP), but a fresh game per ~25 virtual minutes in seed 13 is a cost; the defect part is E1.
Healthy: guard_exit (a safety exit with potions low), recovery (poverty restock when truly broke, `poverty mode: on` 58/30/31 lines), the `xp at risk save point` leaves.

## 6. Early-death rules (item 6)

V: they fired in every seed. Full-log counts: `guard: potion burst (5 drinks in 20s, N hostiles): town portal instead of another potion` 8/5/4; `stale pack noted` 37/24/31; `field gate: stale pack, first field instead` 14/6/9 (`internal/bot/earlyexit.go:46, 64, 92`). Deaths: 0/0/0 (whether the rules prevented any death is U, there is no control run). They are INFO lines, and the ops collect tool keeps only `"level":"(WARN|ERROR)"|failsafe|kill drought|nav stuck|stuck ladder|area abandoned|campaign stalled` (`EXTRAS_KEEP` in the gate judge of the ops repo), so they were filtered. The same filter dropped `merc: revived`, `merc: withdrew`, `merc: gold farm`, `merc: seen` and `nav mouth gate`. Proposed filter change (ops repo, not here): add `merc: |guard: potion burst|stale pack|field gate|nav mouth gate: click held` to EXTRAS_KEEP (about 3,000 extra lines per seed at the g2g counts; the file is capped at 200 KB gzip and drops the oldest lines, so keep `seen` out: it is 460 lines per seed and adds nothing the minute lines do not).

## 7. Ranked fixes

| rank | fix | where | knob / data | effect (U unless noted) | task |
|---|---|---|---|---|---|
| 1 | Merc clock: use the env clock (virtual under lockstep) for revive and death windows; gold-farm need and `need=` use the last READ quote when one exists (fall back to the estimate only with no quote) | `cmd/d2bot/mercwire.go` (`now`), `internal/merc/policy.go` (`ShouldRevive`, `ReviveNeed`), gold-farm trigger | `merc.reviveMaxPerHour`, `merc.stopDeathsPerHour`, `merc.goldFarmMaxLevel` | merc alive time up several-fold in soaks (V that the cap binds every hour); farm-mode confinement down from 34 to 62 percent | TASK-66 |
| 2 | Mouth gate: break the repeated "no safe click" holds (inside-disc detour or an escalation after N holds on the same pair) | `internal/nav/mouthgate.go` `gateRewrite` | `gateMaxRewrites` and a hold counter | removes the F1 ladder time; fewer np_clearing/task_step stalls | TASK-67 |
| 3 | Town portal scroll before gear when the count is 0 and gold covers it; log what was bought when `portal source still empty after restocking` | `internal/town/chores.go` (around line 679) | none | prevents the F2 class (U on why the scroll was not bought) | TASK-68 |
| 4 | Extras filter keeps the merc, early-exit and mouth-gate lines | ops repo gate judge `EXTRAS_KEEP` | regex | evidence only | not a task here (other repo) |
| 5 | Equip failure: do not leave the game; clear the cursor and retry later | leveler equip code | | 7/20/9 fresh games per seed at most | not created (limit of 3) |
| 6 | Merc deaths: a rule that moves leveling to an easier area after repeated field deaths; or, for the Rogue, no field time without an upgrade path (the Act 2 Desert swap is Act 2) | new rule, SME decision | | M3; needs the user | question 3 |

## 8. Needs the user

1. Windows that contain a restart (F1, F2, F3). Question: should the NOKILL and BELT checks ignore a window that contains a fresh-game restart? All 3 failing windows contain one. Recommendation: NO, keep the criterion literal; the restart is the ladder's remedy for the stall, and excluding it would hide exactly the failures. Instead fix the causes (rank 1 to 3).
2. BELT funded test (F3). Question: should a window count as funded by the gold at its first minute instead of its best minute? Recommendation: YES for the judge (a window that starts in poverty and is rescued by a pickup on its last minute is not a failure of the belt logic); this is a judge change, no Numbers line moves. If you prefer the strict reading, F3 is then a revive-before-potions consequence at clvl 8 and you should decide whether revives at clvl under 10 may spend the potion budget.
3. Merc strategy. The Rogue lives 4 to 10 game minutes per revive at clvl 12 to 17 in Black Marsh, Tamoe Highland and Stony Field and is dead 83 to 96 percent of the time in the soaks. Options: (a) fix the clock and quote (rank 1) and accept a mostly-dead merc, (b) keep farming in easier zones until the Act 2 swap, (c) drop the merc from the Andariel interim rule and use the clvl floor alone. Recommendation: (a) now, then re-measure on wave 2; decide (c) only if the merc is still under 30 percent alive.
4. Andariel interim rule. The floor 15 with a live merc is almost never reachable (merc alive 4 to 17 percent), so in practice the floor is 17 with the hold on. Question: keep 17 as the working floor for the 5-seed claim? Recommendation: yes, keep it as written (it is the conservative reading), and note that the pass condition is the hold line, not a fight.

## 9. Open items and dead ends

- Why seed 17 had no scroll at 304 to 460 gold (`Gear: equip and shop` spent 460 to 304 at 20:40:05 and the pack gold went 304 to 47 at 20:41:31 with `portal source still empty after restocking` at 20:41:59): what was bought is not in the kept logs (U). TASK-68 asks for the log line first.
- Whether merc deaths in town are state-read lag or real deaths in the field noticed late: needs a debug trace of the `seen` reads right after `revived`.
- Whether the lockstep wall/virtual ratio (3.5 to 5.0) is the same for the real-time game: it is not, a real game runs at 1:1 and the cap then means 2 per game hour as designed.
- The record's "minute 198 to 205 right after the injected fault" in the brief of this task is wrong (see F3); the record's own table only says "right after the injected kill at the start of the recovery", which is also not what the minute lines show (fault at line 385).
