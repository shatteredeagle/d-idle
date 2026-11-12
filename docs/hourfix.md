# hourfix: first-hour time sinks (TASK-368)

Source: docs/probes/hour1-analysis-2026-11-12.md (fresh Paladin hour, clvl 6, 528 kills) and the bot log of that hour (build 0.0.261112021709). Tags: V read from that log or from a test in this branch, S secondary (SME estimate), U unverified live. Nothing here ran against the game; every minutes figure is a log-derived estimate (U until a new hour is measured).

## 1. Bounce ban took the way out of town (V cause, minutes 8 to 12)
- Cause (V, log 02:46:24 to 02:49:02): leaving the Den lands the hero on the Den mouth in Blood Moor (arrival 145,141); the step-away click (`nav: stepping away from a cave mouth`) was sent every time, and d2d still fired the cave entry within 0.5 to 2.2 s: 6 `arrival bounce` lines, 4 of them for Blood Moor, none passed. So #609/#612 (kick toward the target) did not stop this mouth; it is the same d2d arrival bounce as docs/passagebounce.md, now at a cave mouth beside a hub. U why the first move still bounces (needs a live `nav.trace` look at 50 ms).
- Defect (V): `noteBounceLocked` keyed and skipped the area the hero stood in. Blood Moor is the only way out of the Rogue Encampment, so `routeBan` refused every target (13 picks), the bot started a fresh game and waited 78 s, then re-cleared Blood Moor for 13 min at 171 xp/min instead of Cold Plains at 558.
- Change: `bounceCulprit` / `isTransit` (internal/bot/loop.go). A transit area is one that the mover's route from the act town to another plan step crosses (data: plan steps plus the exit graph, no area ids). If exactly one side of a bounce is a transit, the other side (the cave) is counted and skipped. Both or neither transit: old behaviour. Test `TestBounceAtACaveMouthSkipsTheCaveNotTheTransit`; `TestArrivalBounceSkipsTheBouncedArea` stays (its 3 s stays are now 5 s: a stay under `bounceMaxStay` in the transit side counts for the cave).
- Not fixed (U): the bounce itself. With the cave skipped the hero can still be caught in the mouth loop until a move passes; it is now a 1 to 2 s retry loop that the planner no longer turns into a 13 min detour.
- Class independent. Tested on the shared fake graph (Blood Moor / Den); Paladin and Sorceress share this code path (no class input).

## 2. Idle leveling field (V cause, about 3 minutes)
- Cause (V): Blood Moor 02:24:32 to 02:30:46, 11 kills; `Config.EmptyAreaSec` is 0 by default (docs/killfix.md: 150 s fired 16 times without a kill gain in farm soaks), so a leveling sweep roamed until its explore ran out.
- Change: knob `bot.clearIdleMin` (3, 0 = off, docs/knobs.md). Applies only inside a leveling slice (`taskActs.LevelTo` sets `loop.levelSlice`), through `emptyAreaLeave`; `EmptyAreaSec` and the quest-field limit win when set. Test `TestClearIdleMinLeavesALevelingField`. Farm, gold-farm and drill sweeps are unchanged.
- Expected: about 3 min (6.2 min idle shrinks to 3). U: whether a 3 minute no-kill window ever fires in a field with live packs far apart (the roam walks 40 to 60 s between packs in Blood Moor, V `explore` legs).

## 3. Stale Den claim (V cause, 3 to 4 minutes)
- Cause (V, log 02:38:49 vs 03:03:43): the claim ran in the task layer (`Quest done: den_of_evil`), so `loop.questDone` (set only by `claimStep`) never knew it. The floor planner then picked the Den as a quest run again at clvl 4 and 5, `runFinished` queued a claim, and the claim trip at 03:03 failed ("no skill point after talking to Akara") after a 4 minute re-farm at 223 xp/min.
- Change: `questClaimed` (leveling.go) reads the live quest word (Stage Done) as well as `questDone`; `runFinished` does not queue a claimed quest and `target0` excludes the Den in Normal once claimed. The word survives a fresh game and a restart. Test `TestClaimedDenIsNotARunAgain` (fails without the exclusion, V by revert).
- The area-8 and quest id literals already live in leveling.go (`areaDenOfEvil`, runFor); not new.

## 4. Plan order (V from the table and tests, 2 to 5 minutes)
- Data: Cold Plains gets `melee_enter_clvl` 4 (progression-normal.json and its docs/plan/data mirror). Only `MeleeTank` classes (Paladin, Barbarian) are affected; casters keep 5 and reach it through their Den floors of 5 and 6 anyway (same shape: no change).
- Code: with the Den unclaimed and eligible, `target0` picks the Den before any later step (a melee class at clvl 4 would otherwise take Cold Plains in the overlap of the windows): "Den always taken". With the Den claimed or held, a clvl 4 melee hero goes to Cold Plains; Blood Moor is no longer the last-in-window pick after the Den.
- Tests: `TestPickColdPlainsAtClvl4ForMeleeAfterTheDen` (Paladin, Barbarian, Sorceress), `TestPickColdPlainsNotBelowEntryWhenBloodMoorHeld` updated (melee at 4: Cold Plains; caster at 4: nothing).
- U: that a clvl 4 melee Paladin holds Cold Plains (SME: yes, clvl 4 plus potions; the kit gate `kit_clvl` 4 still applies). The journey spine (data/campaign/normal.json) is unchanged: its chapters already go start, waypoint, Den; the planner only decides what the hero does after.

## 5. Town trips for junk (S/U, 1 to 3 minutes)
- Cause (V): five of the eight town trips were pack-full trips; the hero was in poor mode (gold under `loot.poorGoldBelow` 300), which picks every junk weapon or armor worth `loot.poorMinSell` (6 gold) or more, 4 to 50 gold each.
- Change: `loot.poorMinSell` 6 to 20 (the poor rule is for the rebuy purse; a 6 gold club is not worth a cell and a trip). Test `TestHourfixDefaults`.
- Not changed: identify. The trips of that hour logged `skipped: identify` in their chores (V, 02:48 "chores done ... skipped identify"), so no identify trip for junk was seen; the brief's identify hypothesis is not supported by this log. The sell-trip item minimum (`bot.sellMinItems` 4) and `bot.invFullPct` 85 stay.
- Expected saving: fewer sellable items per fill (maybe 1 to 2 trips, 1 to 3 min). U: no measurement.

## Not done
- Stall pauses (311 s) and the killtime wait split: measurement tasks from the analysis, no change here.
- Live check list for the next hour: (a) `arrival bounce` lines count and the `area skipped` reason (should name the Den, never Blood Moor); (b) `area abandoned: no kill for a while` in Blood Moor near minute 5 to 8; (c) no `claim den_of_evil` after the first claim; (d) first Cold Plains entry before minute 35; (e) town share of the hour under 8 percent.
