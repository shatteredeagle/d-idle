# Gate 2 run 20261102: failure analysis

> status: current analysis of run gate2-20261102-0823 (record: docs/soaks/gate2-20261102.md on branch agent/gate2run-20261102). Numbers are owned by openspec/specs/soak-gates/spec.md; this note proposes no spec edit, only a possible delta (section 6). Tags: V = counted from the run's logs on the worker host (read-only), S = from code or a doc, I = inference, U = unverified.

## 1. The run in one paragraph

Fresh Sorceress, seed 11, binary `d2d.lab2-freezeresume` (fork `ee5e06d8ef99`), bot `e5e0b8c`. 480 wall minutes = 2811 virtual minutes, clvl 16 reached at virtual minute 2223, 16,398 kills, 0 deaths, 0 panics, 0 bot restarts, the one injected kill recovered in 2 s (V: `state`, `logs/supervisor.log`, `result/compare.txt`). Five criteria failed: NOKILL, BELT, INVARIANTS (the first two), no-gain window, merc hire. XP/h 5244 passes the 1000 floor. d2dsoak groups the run `VIOLATED(BELT,NOKILL)`.

## 2. Verdict per failed criterion

(a) real bot or d2d defect, (b) CPU starvation artefact, (c) gate-rule or judge issue. Virtual minute = one `minute` line.

| criterion | what tripped (V) | verdict |
|---|---|---|
| NOKILL: 15 min (limit 10) | virtual minutes 864 to 878 (wall 09:43:52 to 09:44:23, 31 s wall = about 29x speed, not starved). The bot was in town chores (Rogue Encampment): waypoint 4 to 1, two vendor visits, 27 `shop: bought` lines, a `nav stuck` in town, no kill and no XP. Two shorter stretches of 8 (vmin 481, vmin 1289) are also town or travel. | (a) with (c). The resupply takes 15 virtual min because every potion is a separate vendor click; the rule counts town time as "no progress" although the state is a legitimate chore. Not (b). |
| BELT: 10 min (limit 5) | virtual minutes 432 to 441 (wall 09:05:18 to 09:05:51), clvl 11, Cold Plains, state `recovery` ("Recovery: farming gold"), gold 33, belt had only mana potions. The belt healing fell 4, 3, 2, 0 within seconds while `combat abort fight: flee` fired 30 times and `potion: not drinking because chose crowd-retreat first` 59 times. At 09:05:02 the kit trip fired as an emergency ("kit: healing potions 0/4"); the town portal was cancelled ("d2d portal: cancelled, nothing clicked (bot: safety crowd-retreat)"), the bot fell back to walking home, and the 10 minute lines outside town were that walk. | (a). Poverty (gold 33 at clvl 11) plus a kit trip that is triggered at 0 healing potions, not before, and that the crowd-retreat can cancel. Not (b): speed was 7 to 9x. |
| d2dsoak INVARIANTS | Only the two above (`VIOLATED(BELT,NOKILL)`); hang 0, stall 0, wedge 0, deaths 0, restarts: fresh 41, chickens 0, bot 0. | follows its parts |
| No-gain window 747 min (limit 90) | virtual minutes 1475 to 2222. The judge counts a gold gain only above the all-time high-water mark (1485 reached at about vmin 1475), but gold is spent in town every few minutes, so no later gain exceeds 1485 until the very end. Real gold gains happen all the time (per-200-minute maximum gold between 835 and 1485). The clvl did not rise: clvl 15 at vmin 1263, clvl 16 at vmin 2223 (960 minutes). | (c) for the 747 number (judge artefact: the spec says "no clvl, quest step or gold gain", S); (a) for the underlying fact: the clvl 15 to 16 plateau of 960 min is real and is caused by the merc defect below (I). |
| Hire the merc: 0 lines | The judge looks for "hire(d) merc" / "merc hired" (S: `tools/gate-judge.py` in the ops repo). The bot logged `merc: claimed free rogue` at 08:31:09 right when `Quest done: sisters_burial` (clvl 7). Five seconds later the route guard reads `merc=none` and stays `none` for the rest of the run: 4 lines `floor=16 (merc=none)`, 2 `floor=17`, 19 `guard: leaving guarded area ... (no merc, clvl < 16)`. Only 1 line `merc=alive` ever (08:31:09, same second as the claim). | (a) and (c). The Normal-act-1 policy (S: `cmd/d2bot/acts.go`, plan-v2 10.1) is "never pay Kashya, claim the free Rogue"; the claim reported success but no merc was readable afterwards, and nothing re-checks or retries (U: whether the claim never granted a merc or the reader lost it; `claimDep` hard-codes `AtKashya: true` and `BloodRavenDone: true`, S: `cmd/d2bot/mercwire.go`). The judge wording does not match the free claim either (c). |

Also found while checking (judge, not a failure): "Attempt Andariel at clvl 14 or higher: PASS, 66 lines" is a false pass. The 66 lines are `planner` lines whose text mentions Andariel (V: 36 Crypt, 15 Jail, 15 Catacombs "why" strings); the real boss lines are one `boss: held for andariel (unattended hold ... boss.andarielHold)` and `route guard: andariel floor=17 (merc=none)`. Andariel was never attempted (V).

## 3. The 41 restarts

The counter is not d2d or bot crashes. V: d2dsoak says `fresh41 / chk0 / bot0`, chickens 0, supervisor.log shows one launch and one fault. Cause breakdown from `bot-0.log`:

- 34 lines `leveling: everything worth farming is cleared, starting a fresh game` (bot-started new games, design, S: `internal/bot/goal.go`),
- 1 injected kill (09:23:39, `game lost, restarting it`),
- the rest are double counts: four increments of +2 (22 to 24, 26 to 28, 30 to 32, 32 to 34) are a fresh game plus a save-point re-entry in the same stretch (same effect seen in Gate 1, whose record notes 2 such cases; U why).

Why it accelerates: from vmin 1264 (clvl 15) the character is blocked from the Monastery, Forgotten Tower and Andariel by the route guard (no merc: floor 16 and 17, S), so the farm list is the act 1 fields only. Each pass clears 3 to 10 areas, the sweep reports `cleared=true`, and the bot starts a fresh game to respawn them. Restarts per 500 virtual minutes: 5, 4, 7, 10, 12, 3. The 3 in the last block is the end-of-run short block. Several lines show `cleared:3` or `cleared:4` after only 35 to 150 s wall (11:47:09, 11:47:44, 11:47:57): V that the fresh games follow quickly in the slow stretch, I that skipped areas (232 `area skipped`, 10 `area abandoned: ping-pong without a kill`) shrink the list faster under load.

Memory: `mem.csv` (d2d RSS, wall) peaked at 838 MB, no growth trend visible over 480 min and no OOM (V; the 27 MB minimum is just after the kill). No stuck state: 0 hang, 0 stall rungs, 0 wedge.

## 4. CPU starvation (b)

V from the minute timestamps (virtual minutes per wall minute, 60 minute blocks): about 17, 15, 15 in the first 180 minutes, 7 to 12 until vmin 1260, then 4.0, 10.2, 1.3, 3.1, 4.3 for vmin 1260 to 1560, and 3 to 6.6 for the rest. The slowest 10 minute chunks are around vmin 1390 to 1440 (5 to 7x per chunk, 1.3x over the block). So the speed did fall, longer and further than "15 minutes at 1x" (the whole second half ran at 3 to 6x instead of 10x). But the two violation windows sit at vmin 432 (speed about 9x) and 864 (about 11x): neither is a starved stretch. The lockstep clock makes the bot's behaviour independent of speed except for wall-time timers (ping-pong guard 735 to 1194 s, linkwait); the late-run speed fall can therefore only affect the ping-pong abandon count (I), not the failed criteria. Verdict (b): none of the five failures is a starvation artefact. The slow stretch should still be kept off the next run (section 5).

## 5. Gate 1 versus Gate 2

| item | Gate 1 (bot `050a509`, 977 vmin) | Gate 2 (bot `e5e0b8c`, 2811 vmin) |
|---|---|---|
| NOKILL longest | 5 | 15 (at vmin 864, inside the first 977 min) |
| BELT longest | 4 (1 minute margin) | 10 (at vmin 432, inside the first 977 min) |
| Criteria only in Gate 2 | n/a | no-gain, merc, clvl 16, Andariel, INVARIANTS (the Gate 1 judge lacks them) |
| Restarts | 11 at 977 min (about 1 per 89 min) | 41 at 2811 min (about 1 per 69 min) |

Both Gate 2 violations fall inside the first 977 minutes, so they are not a length effect. The bots differ by the new kit gate (V from `git diff 050a509 e5e0b8c -- internal/bot`: kitgate.go +109 lines, goal.go, recoveryxp.go, resource.go, tpdebounce.go): `KitGateLevel 8` opens the band where an incomplete kit becomes a town trip, capped at `KitTripCap 2`, with `KitMinHP 4` and `KitMinMP 3` (S: `internal/bot/kitgate.go`, `resource.KitMinHP`, `resource.KitMinMP`). Logs of that gate in Gate 2: 118 `kitgate-trip`, 35 `kitgate-skip`; 10 `kit gate on` and 10 `off` (the first pairs at clvl 1 to 3 within the first 4 wall minutes; U for the later ones). The late trips are mostly "mana potions 0/3": about 10 trips between 15:41 and 15:57 wall (V) show the trip loop hitting the cap (2 trips, skip, reset when the kit is complete, again). The BELT event is exactly the new path (an emergency kit trip at clvl 11 with 0 healing potions). I (not proven): the same seed 11 diverges with a different build, so Gate 1 may simply not have met the poverty plus crowd situation; the belt threshold in the trip is 0 healing potions for the emergency while the kit minimum is 4.

Positive: the kit gate itself held below clvl 8 (0 deaths in 16,398 kills, about 2.7 times Gate 1's kills), and the first holds were released within seconds (V: 08:23:42 on, 08:24:31 off; 08:25:27 on, 08:25:30 off).

## 6. Fixes, ranked by value

1. **Keep the free Rogue (bot, highest value).** Why: it removes the Hire criterion, the Andariel block, the route-guard floor of 17 (merc=none) versus 15 (merc=alive), the 960 minute clvl 15 to 16 plateau (the no-gain "real" part) and most of the restart acceleration. Paths: `cmd/d2bot/mercwire.go` `claimDep` and `Claimer.ClaimFreeRogue` (verify the merc after the claim with the reader instead of assuming `granted`; log `merc: claimed` only after `rd.State()` shows a merc; retry at Kashya if not), `internal/task` claim step `quest.sisters_burial_merc/claim` and `task.MercAlive` (why `merc=alive` once then `none`), `cmd/d2bot/acts.go` (the act 1 hire is refused by design; the claim must be the only path). Needs a short live check on a slot (checklist below).
2. **Emergency kit trip before the belt is empty and when the portal is cancelled (bot).** Why: removes the BELT 10. Paths: `internal/bot/kitgate.go` `kitTrip` (start the trip at `HP < kitMinHP` while the way home still works; do not let `crowd-retreat` cancel the only portal attempt: retry once after the retreat or walk to the nearest waypoint), `internal/bot/death.go` and `goal.go` (the emergency trip), `internal/bot/tpdebounce.go`. Also stop recovery gold farming in Cold Plains at clvl 11 with 33 gold and no healing: a poverty sell trip first.
3. **Shorter town resupply (bot).** Why: removes NOKILL 15. Paths: `internal/town` shop (27 single buys in one visit: buy the planned stack in one click or in fewer vendor round trips), the `nav stuck` in town at the vendors. Alternative or addition: a rule delta (below).
4. **Judge fixes (ops tool `gate-judge.py`, not the spec).** (i) no-gain: count gold as a gain when the minute-to-minute gold rises, not only above the high-water mark (this matches the spec text "gold gain"); (ii) merc: accept `merc: claimed free rogue` followed by a readable merc, or the hire line; (iii) Andariel: match the boss lines (`boss: ... andariel`, `route guard: ... andariel`) not the `planner` why strings. Without these the next record would repeat the false results.
5. **Fresh-game loop at the cap (bot, lower).** With a merc the route opens; without it, a clvl 16 character should not burn 40 games on the same cleared fields. Paths: `internal/bot/goal.go` (leveling fresh game). I: not needed once fix 1 lands.

Possible rule delta, for the user (U that it is wanted): soak-gates "No single state over 10 min without progress" could exempt `town chores` / `resupplying` up to a stated length (for example 20 min), because a legitimate resupply is not a stall; if kept as it is, fix 3 is required. No number is proposed here.

## 7. Re-run plan

1. Land fixes 1 and 2 (and 3 or the rule delta), and the three judge fixes; merge to public main; a pre-run check that the CPU is free: no niced agent builds on the worker host during the run (`labrun the worker host -- top -bn2 -d1`, idle plus nice at least 25 percent at start and the slot reserved; the run loses its 10x speed otherwise, section 4).
2. Re-run Gate 1 first on the new bot commit (spec: Gate 2 on a build whose Gate 1 did not pass does not count, S: soak-gates "Skipping a rung"): seed 11 again for comparability and one new seed for coverage; 2 h each.
3. Then Gate 2 with seed 11 and a second seed, 8 h, fault at 60 min. Success shape: merc readable after the claim in the first 300 virtual minutes, BELT at most 5, NOKILL at most 10, no plateau over 90 minutes at clvl 15.
4. Live check for fix 1 (checklist, nothing run by this note): on one slot with a character that has killed Blood Raven, run the claim step with `merc` wiring on, read `merc.name` and `merc.level` through the d2d reader before and after, and confirm the Rogue follows through a town portal. U until done.

## Shared-file lines for the merger

In `openspec/changes/gate2analysis/merge-lines.txt`.
