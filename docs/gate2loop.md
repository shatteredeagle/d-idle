# gate2loop: Gate 2 seed 13 save-point loop

Run gate2-s13-20261103-2211-g2v, bot log read offline on the worker host. Tags: V = log line or code read, U = unverified.

## Log evidence

- V: 789 `save point: saving and re-entering the game` lines in 176212 log lines. The first at clvl 3 (22:13), healthy ones
  every 20 virtual minutes, each with `xp at risk` > 0 or a short-lived 0.
- V: the first no-target block `no eligible reachable leveling area for level 15` is at 23:36 (line 41596). The first
  `item stuck on the cursor: hand: no free space in target` (loot pick-up) is at 23:57 (line 54166).
- V: from 23:39 (clvl 15) save points come 20 to 40 wall seconds apart, later 9 s apart (lines 175915, 176042, 176171).
  Wall time is short because the virtual clock jumps: the 5 min self pause ended in under 3 s wall, bans of "3 more min" run
  down within 0.05 s.
- V: one late iteration (lines 175915 to 176042): save point at clvl 16 with `atRisk: 0`; re-entry; `Recovery: farming
  gold` (gold 26); area gate refused (no leveling area, Blood Moor banned); loot fails five times with the cursor error;
  failsafe ladder steps 1 to 5; `Failsafe: paused`, `self pause ended by timeout`; `recovery: leaving Blood Moor first`;
  `d2d town portal: no portal source (empty)`; `no town portal: walking back to town`; `Entered Rogue Encampment`; and 6 ms
  later `save point: saving ...`. No chores line, no shop line in between.
- V: earlier (23:39 to 23:56) the same shape ran with the tome counting down 5, 4, 3, 2, 1, 0 and one real shop visit
  (`Scroll of Town Portal` x5, 500 gold, line 44933); gold then fell below the 300 gold floor and stayed there.

## Root cause

- V (code): `runGoal` called `savePointStep` right after the trip check and before `beltFillStep`, `needChores`,
  `gearStep` and `claimStep`. `savePointDue` allows a save when the character is in town, no trip is running and
  `now - inGameAt >= SavePointMin` (20 min). Walking home with `ToTown` outside a trip leaves `l.trip == nil`, so the very
  first town tick after the walk-back saved and re-entered the game instead of doing chores.
- V (code): the 20 min test is virtual time, which advances through pauses and bans, so it is satisfied again in a few wall
  seconds. A re-entry lands where the game puts the character (town), the recovery step walks out, hits the same fault, and
  the cycle repeats. Seed 11 reached clvl 16 without this because it never hit the no-area plus cursor-fault combination (U).
- V: the save point protected nothing: `atRisk` was 0 at every late one (no XP since the entry).
- U: the underlying stall (leveling blocked at clvl 15 to 16, then the cursor fault in gold recovery) is not fixed here.
  Chores might not have helped either (3 sellable items below `SellMinItems`, 26 gold), so the loop is a symptom
  amplifier, not necessarily the only cause of the missing XP.

## Fix

- `internal/bot/goal.go`: the save point moved after the chore decision (trip start, heal, gear, claim). A due trip starts
  first; once it is done, the cooldown keeps it from restarting and the save point can run.
- `internal/bot/savepoint.go`: `savePointDue` returns `nothing to save` when the observer's XP at risk is 0 (same figure as
  the `xp at risk` log). A game that earned nothing since its entry is not re-entered again. Items and gold alone no longer
  trigger a save point (U: accepted, a crash loses only what was picked up in a game that earned no XP).
- No threshold, skill or class literal changed.

## Tests (fake, no game)

`go test -count=1 -p 1 -run 'SavePoint|OtherGame' ./internal/bot`

- `TestSavePointWaitsForTownChores`: in town after 25 virtual minutes with XP at risk and a forced chore trip, `runGoal`
  must start the trip and not call NewGame.
- `TestSavePointNothingEarnedIsNotDue`: XP at risk 0 gives `nothing to save`.
- Existing save point tests now earn some XP in `spRig` first. Not run against the old code (the revert check was not
  possible in this runner): by construction the old order calls the cycler first and the old due check has no
  `nothing to save` reason.

## What a live rerun must show

Soak with seed 13, same settings, no NOKILL:

- save point lines spaced at least 20 virtual minutes apart, each preceded by `xp at risk` with `atRisk` > 0;
- `chores done` / `shop: visit` before any save point after a `no town portal: walking back to town`;
- no run of `save point` lines with `atRisk: 0`;
- XP keeps rising past clvl 16. If it does not, the cursor fault in gold recovery and the no-area block at clvl 15 to 16 are
  the next suspects (separate change).
