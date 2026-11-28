# Fable review 2026-11-28, part 13: both sandbox heroes after the fix7 deploy (live logs, read-only)

> status: review, read-only. Window: 2026-11-28 05:19:03 to 05:54:28 UTC (00:19 to 00:54 EST, about 35 minutes), plus the three hours before it as the baseline. Sources: `d2bot ops log --tail 200000`, `ops status --why`, `ops quest status`, `docker logs --since 3h` in the two sandbox containers. No order was sent, nothing was restarted, no game was touched. `ops log` has no `--since` flag (the brief's form is refused: "flag provided but not defined: -since"); the window was cut by timestamp. Tags: **V** read in the log or the code at the cited place; **S** reasoned, not run; **U** unproven. Log times are UTC (EST is UTC minus 5 hours). Code paths are at tree `b37c8f651`.

## 0. Verdict

Both heroes progress, nobody died, nothing hung, no crash. The deploy reached only ONE hero. Doofer, now steered by the arbiter with all families, shows four behaviour classes that were not in its log before the deploy; two are HIGH and none of them is in the N1 to N6 list the fix7b worker has.

| | Doofer (`d2play-20261108`) | Questy (`d2play-20261125q`) |
|---|---|---|
| build (V, `d2bot start`) | `0.0.261128051841`, started 05:19:03, commit `1f5f0f8` | `0.0.261128020453`, started 03:41:17: NOT redeployed |
| arbiter (V, `ops status`) | `bot.arbiterControl` (1), `bot.arbiterFamilies` (255) | `bot.arbiterControl` (0), `bot.arbiterFamilies` (3), shadow only |
| hero | Paladin clvl 18 | Paladin clvl 17 |
| XP per hour in the window (V, first and last `tick`) | 6774 XP in 34.9 min = 11.7 k/h | 9588 XP in 35.5 min = 16.2 k/h |
| XP per hour, 3 h before (V) | 11.3 k/h | 16.9 k/h |
| against the earlier figure | above 5.8 k (U: which hero the 5.8 k and 18.9 k belong to is from the brief, not re-derived) | a little under 18.9 k |
| deaths since the deploy | 0 (death report: 1 in total, 2026-11-27) | 0 |
| lowest life seen | reflex drink at 49 percent, 05:46:24 | emergency portal at 128/243, 05:33:18 |
| log gap above 8 s | none | none |
| WARN classes new since the deploy | `failsafe ladder` (2), `failsafe: skipped the stalled target area` (1), `nav: stalled in a pocket, escaping by town portal` (2); `leveling area unreachable` 12 against 3 in the 3 h before | none new |

The 35 minutes are a short sample: one XP figure per hero, no difficulty hop, no gear change. "Clean" below means "nothing seen", not "proven".

## 1. Findings, ranked

### L1 (HIGH, Doofer, new since the deploy). One cut-off cell bans every leveling zone in 80 seconds

Evidence (V, Doofer log):

- 05:47:53 `leveling area unreachable area=21 "Tower Cellar Level 1" err="exit to Tower Cellar Level 2: nav: no path: reached the closest reachable point {14 133} to {10 140}"`. The hero stands in Tower Cellar Level 3 and cannot reach the stairs up.
- 05:48:01 to 05:49:13: the same error text from the same cell (`gate to level 22 is on the map but not reachable from {15 133} (cause=cut_off, no way out found)`) is charged to a new target every 8 to 9 s: Tower Cellar Level 2, Tamoe Highland, Stony Field, Cave Level 2, Dark Wood, Black Marsh, Cave Level 1, Blood Moor, Cold Plains. Ten `area skipped ... reason=unreachable`, 150 s each.
- 05:48:29 `failsafe ladder step=1`, `failsafe: skipped the stalled target area "Cave Level 2" reason=nav-stuck`; 05:49:13 `failsafe ladder step=2`, state "Failsafe: resetting in town". The last `minute` line still shows `rung` 2.
- 05:49:25 `zone choice: Tower Cellar Level 4 margin 1.9 (need 1.5; low 1.4) ... why: hole_first_field` (finding L2).

Cause (V code, S link): `internal/bot/task.go:1178` to 1184 skips the TARGET for any travel fault. The fault here is a property of where the hero stands (the first leg, out of the current level), not of the target. Nothing in the loop asks "did the last N failures start from the same cell".

Not covered by fix7b: N1 to N6 are the Report back-off, the heartbeat, the survive fact, the first-field floor, the band rule and corpses.

Do: classify the failed leg. When the leg that failed is the exit of the CURRENT area, do not skip the target; leave the pocket (the `nav: stalled in a pocket, escaping by town portal` rung exists, V 05:20:53) and pick again from town. Two equal errors from the same cell are one fact, not two failures.

Must-fail test (`internal/bot`, fake nav, Paladin and Sorceress): hero in area A, `travelTo` returns "no path" for the exit of A for every target; run five slices; assert at most one zone is skipped, no `failsafe ladder` step is counted, and the second slice is a town portal.

### L2 (HIGH, both heroes). The "easiest field" of a hole is the hardest zone left, and the hero walks to it

Evidence (V):

- Doofer 05:49:25, after L1 banned every safe zone: `zone choice: Tower Cellar Level 4 margin 1.9 (need 1.5; low 1.4) k 3.00 xp/min 20 ... hole_first_field (nothing is safe enough: the easiest field, and the hero needs help) | not Jail Level 1: margin_low`. The hero then took the waypoint and went down: 05:50:13 Cellar Level 1, 05:50:45 Level 2, 05:51:07 Level 3. It stopped only on a nav fault: 05:51:09 `exit to Tower Cellar Level 4: nav: no path ... (cause=exit_trigger: every way to a connected cell passes an unwanted exit's trigger area)`. Level 4 is the Countess; the board had PARKED that chunk at 05:25:02. The leveling family walked to the boss the quest family gave up on.
- Questy (older build), 58 times since 02:05: `zone choice: Catacombs Level 3 margin 0.0 (need 1.5; low 0.0) ... hole_first_field ... | not Catacombs Level 2: margin_low, margin 0.0`, then `leveling target not reachable, picking another ... 11 hops and no activated waypoint on the way`, then the same for Catacombs Level 2, then a normal `best_pay_eligible` pick of Tamoe Highland with margin 1.7 to 2.1. So "nothing is safe enough" was false; a clvl 17 hero is kept out of Catacombs Level 3 only by a missing waypoint. Doofer's old build did the same 51 times (margin 0.1); its new build logs `zone candidate dropped: Catacombs Level 2 not reachable now` first and has no such pick (V), so this half looks fixed by the new build and Questy simply does not have it.

Cause (V `internal/survival/controller.go:256` to 266): with no eligible zone the pick is the highest corrected margin among the candidates that are LEFT. After bans (Doofer) or after the band cut (Questy, S: the candidate list in the line holds only the two Catacombs levels) the leftovers are the dangerous zones. `Decision.Hole` and `NeedsYou` are set, and the runner travels anyway.

Do: a hole decision never travels to a zone whose margin is under `need`. With a hole the hero stays in town or goes to the act's first field; zones banned only for `unreachable` count as candidates again before a margin-low zone is taken. A parked boss chunk removes its area from leveling candidates.

Must-fail tests: (a) `survival.Choose` with candidates {Jail Level 1 margin 1.1, Tower Cellar Level 4 margin 1.9}, need 1.5: assert the decision names no travel target (or the first field) and `NeedsYou`; (b) bot level, two classes: every eligible zone carries an `unreachable` ban, one margin-low zone is free: assert the runner does not call `travelTo` for the margin-low zone; (c) `forgotten_tower/countess` parked: assert Tower Cellar Level 4 is not a leveling candidate.

### L3 (HIGH for quests, Doofer). The exit-trigger gate traps the hero next to stairs; three retries from one cell park the chunk

Evidence (V): 05:24:18, 05:24:31, 05:24:45 `quest board step ... exit to Tower Cellar Level 4: nav: no path: gate to level 24 is on the map but not reachable from {45 21} / {42 21} / {42 21} (cause=cut_off, no way out found)`; 05:25:02 `quest board: parked forgotten_tower/countess why="gave up after 3 tries"`. Three tries in 27 s, the hero did not move between them. `ops quest status` now shows `forgotten_tower/countess blocked auto=on code=no_path`. New lines since the deploy: `nav mouth gate: click clipped at a read trigger area` (4), `combat swing held: the click would walk into a cave mouth` (1), `combat move skipped: the point is in a cave mouth` (repeated 17 times in 3 s at 05:46:58), `nav: stepping away from a cave mouth` (22). The same shape produced L1 ({15 133}) and the fault at {15 147}.

S: the hero fights or is pushed into the trigger area of an exit it does not want, and from there every path "passes an unwanted exit's trigger area", the start cell included. The round-2 review lists the mouth gate as still open (R3-5); the live cost is a parked quest and the L1 cascade.

Do: the start cell and its neighbours are exempt from the unwanted-exit penalty (the hero must be able to walk OUT of a trigger area); a quest retry that starts from the same cell within a minute is not a new try: step out or portal first.

Must-fail tests: (a) `internal/nav`: grid with two exits, start inside the trigger area of exit A, goal exit B: assert a path exists; (b) quest board, two classes: three `no path` results from the same cell inside 60 s count as one attempt and the chunk is not parked.

### L4 (MEDIUM, Doofer, new line since the deploy). A trip held for a reward nobody claims, 300 log lines

Evidence (V): `trip held: a quest reward is pending, the board walks home first` 300 times between 05:19:06 and 05:20:53 (about three per second, `internal/bot/goal.go:376` has no throttle). In the same window every `tick` carries `claim_pending` 0 and `plan_buy` 2, and the hero did not walk home: 05:19:10 Black Marsh, 05:19:50 Forgotten Tower, 05:19:53 Tower Cellar Level 1. The hold ended by accident at 05:20:53 (`nav: stalled in a pocket, escaping by town portal`). The pending reward was `search_for_cain=0x201e`, a chunk the board lists as `auto=never`; it was collected in passing at 05:30:12 while buying potions (`search_for_cain=0x201d`, `Quest done: search_for_cain`).

S: `holdForReward` asks the quest words, the arbiter fact asks the board, and they disagree for an owner-only chunk. A non-emergency trip can be held for as long as the hero stays out of town.

Watch on Questy: Search for Cain is the owner's quest there (`search_for_cain/scroll ready auto=never`, word `0x0004`). The same passing visit to the potion seller will take the reward once the owner has freed Cain. U whether the owner minds; say so before it happens.

Do: hold only when the board has a claim step it will run (the `claim_pending` fact); log once per hold.

Must-fail test (two classes): reward word pending for an `auto=never` chunk, hero in the field, plan with a buy line: assert `needChores` reports the trip due and at most one "trip held" line is written in 60 fake seconds.

### L5 (MEDIUM, arbiter control). The travel budget counts a finished town trip as "nothing progressed"

Evidence (V): Doofer 05:38:14 and 05:38:47 `arbiter shadow diverge: ... quest/step: travel budget: going to Tamoe Highland undoes Tamoe Highland->town 0s ago and nothing progressed since`; the trip in between repaired gear at 0 percent durability and spent 1802 gold. With control on this is not only a shadow line: 05:38:48 state "Leveling: the arbiter holds the quest step", and the level family picked Cave Level 2 (163 XP per minute) while the board step was running Tamoe Highland. Again at 05:44:01 and 05:44:32 (Cave Level 2). Questy logs the same refusal in shadow (05:20:05, 05:48:59, 05:49:23).

Cause (V `internal/arbiter/arbiter.go:321` to 324): the refusal compares `v.Progress` only; chores done, gold spent and potions bought are not progress. Not N1: N1 is the `Report` back-off key.

Do: a completed upkeep intent (trip reported true) clears `a.last`, or counts as progress.

Must-fail test (`internal/arbiter`): view in field F, upkeep chosen, travel F to town recorded, `Report(true)` for upkeep; next view in town with a quest intent `NeedsArea` F: assert it is chosen, not refused with "travel budget".

### L6 (MEDIUM, both). `arbiter-starved` fires on a step that IS leveling

Evidence (V): Doofer 05:29:38, Questy 05:31:03 (and 03:51, 04:55): `invariant arbiter-starved detail="level/sweep: nothing else wanted: level the hero; outranked by quest/step (quest step in Tamoe Highland)"`. The decision ring names the winning step "Quest board: training" (`ops status --why`), state "Quest: fighting (Level up to 19)". The filler is "starved" by an objective that does the filler's own work. One false soft hit per 10 minutes of training; it will hide a real starvation.

Do: an intent is not starved while the chosen intent's executor is the same activity, or while XP rose in the window.

Must-fail test (`internal/bot/invariants_test.go` style): 700 fake seconds of quest/step with step kind training and rising XP: assert no `arbiter-starved` hit; with a real foreign step and flat XP: assert one.

### L7 (MEDIUM, Questy, item flow). Vault full: rare and magic drops stay on the ground

Evidence (V): 23 loot lines with `vault full` in 35 minutes, among them `rare Heavy Gloves` (05:54:28), `magic Ring` (twice), `magic Amulet`, `magic Small Charm`, `superior Hunter's Bow`. The pack is nearly empty (`invPct` 17), `loot.vaultAutoTrim` (0). The shadow rules would pick them up (`rules=pickup rule=g-magic-rare`, `g-small-charm-id`). Doofer: none in the window.

Do: with the vault full, an item the rules call pickup falls back to carry and sell (or keep in the pack) while pack cells are free; raise a "needs you: vault full" line once.

Must-fail test: vault at capacity, pack at 17 percent, ground item magic Ring: assert the pick decision is "pick" with destination pack, not "skip: vault full".

### L8 (LOW to MEDIUM, Doofer). Gear reaches 0 percent durability before the repair trip

Evidence (V): 05:38:14 `town trip reason="[heal equipment at 0% durability]"`, repair 1763 gold; at 05:30:27, eight minutes earlier and in town, `repair (not needed)`. Fourth such trip in the log (23:27, 01:09, 02:19, 05:38). Knobs: `town.repairBelowPct` (30), `town.tripRepairPct` (20). S: an item with a small maximum (the worn line shows one slot at 5 of 15) passes from above 30 percent to 0 inside one outing, or the percent is taken over the whole set. A broken weapon is a silent damage loss for a melee hero.

Must-fail test (Paladin and Amazon): worn weapon at 4 of 15 durability, hero in town on any trip: assert `repair` is in the chore list; in the field at 2 of 15: assert a trip is due before 0.

### L9 (LOW)

- Empty sweeps: Doofer swept Tower Cellar Level 1 and 2 with `kills 0` three times (05:28:50, 05:52:42, 05:53:57), 70 to 90 s each; the zone value (`realized 355`) does not know the level was emptied while walking through it in the same game.
- `class-aware start class=Paladin combatProfile=paladin combatBuild=blizzard` (V, both builds, `cmd/d2bot/class.go:84`): a Sorceress build name on a Paladin. U whether it is only a label; it is the silent default the build rules forbid.
- `tick` lines are written at action boundaries only (80 in 35 minutes) and their `area` is the area of the previous boundary (05:23:14 says "town" with the hero in Tower Cellar Level 3). Anyone reading facts from `tick` reads stale ones.
- Questy's old build logs a save point as `WARN d2d: Save and Exit requested (chicken / leave game)`; the new build says `(savepoint)` at INFO. Gone with a redeploy.
- Questy 05:42:40 `xp at risk why="save point" atRisk=8741` against 3281 on Doofer: about half an hour of XP unsaved with `bot.savePointMin` (20). S: the timer restarts on something other than the last save.

## 2. The questions of the brief

- **Progressing?** Yes, both (table in section 0). Doofer lost about 6 of 35 minutes to L1, L2 and the empty sweeps.
- **New warn or error classes since the deploy?** Doofer: the failsafe ladder pair and the pocket escape (L1, L3); no ERROR line on either hero. Questy: none (it has no new build).
- **Arbiter intent oscillation?** Not seen. 22 decision-ring entries in 35 minutes on Doofer, the longest 21 ticks; nine shadow divergences, all explained by L5 and by chores ordering. The one control artefact is L5 (quest step held after a trip).
- **Hang or stall (N2)?** Not seen: no log gap above 8 s on either hero, no supervisor restart after 05:19:03, no watchdog line. N2 is neither confirmed nor refuted by this window; the ladder that did fire (L1) was fed by action failures, not by a hung slice.
- **Deaths?** None. Nearest: Doofer two reflex drinks at 54 and 49 percent in Tower Cellar Level 3 (05:46); Questy an emergency portal at 53 percent with a stale pack of six (05:33:18), handled in 20 s.
- **Flip windows.** `bot.diffReadyV2` (1): only the refusal line, coherent on both ("not ready for nightmare: Baal not dead ... clvl 18, need 38 ..."); no go, no hop. `items.advisor` (1): one `gear advice` line at start with `hints: null` and the text "Your hero wears its starting gear" for a clvl 18 hero with nine worn slots (U whether the text is a default). Neither window was exercised; nothing breached. Admission shadow: 1580 and 1593 agree, 0 differ.
- **Quest board on Questy.** `search_for_cain/scroll ready auto=never`, the four later chunks locked: untouched by the bot in the window (V: no cain line). `tools_of_the_trade/malus blocked code=handin_failed`, no retry in the window. `forgotten_tower/countess done`. No park, no resume in the window. On Doofer: `forgotten_tower/countess` auto-started 05:20:58, parked 05:25:02 (L3); `tools_of_the_trade/malus ready auto=on` (word 0x0004 then 0x000c at 05:38).
- **Gold, potions, belt.** Doofer: pack 1240, stash 254101, belt 6 healing, 4 in the pack, tome 15. Questy: pack 1546, stash 13330, belt 3 healing and 2 mana, 1 healing in the pack, tome 2 charges (the plan line says `portal 1/1`, so it will not buy until 1): thin for a hero that used an emergency portal 20 minutes ago.

## 3. What fix7b will not cover

L1, L2, L3, L4, L5, L6, L7, L8. Only the "quest step held" symptom of L5 touches the arbiter code fix7b is in, and it is a different rule (travel budget, not back-off).

## 4. Watch at 06:00 EST

1. Doofer: count of `failsafe ladder` and `leveling area unreachable` since 05:54 UTC; any `hole_first_field`; whether `rung` in the `minute` line fell back from 2.
2. Doofer: `forgotten_tower/countess`: still `blocked no_path`, or retried and parked again (L3); any zone choice of Tower Cellar Level 4.
3. Doofer: `trip held` line count (L4) and `arbiter-starved` count in `ops status --why` (L6); state "the arbiter holds the quest step" durations (L5).
4. Both: deaths, supervisor restarts, log gaps above 60 s (N2), XP per hour against 11.7 k and 16.2 k.
5. Questy: portal tome charges (2 now), belt healing count, `vault full` lines (L7), `xp at risk` at the next save point.
6. Questy: `search_for_cain` word (0x0004 now) and any `Quest done: search_for_cain` written by the bot (L4 watch); `malus` still blocked without thrash.
7. Whether Questy was redeployed (its `d2bot start` version); until then it says nothing about fix7.

## 5. Not done (U)

No code path was run; the causes in L1 to L5 are read from the cited lines only. The whole night before 02:19 UTC was not read. Which commit build `0.0.261128020453` contains was not checked (it decides whether the Questy half of L2 is a fix3 regression or older).
