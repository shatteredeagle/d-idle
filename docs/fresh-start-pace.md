# Fresh start pace: why the tester hero crawled in Blood Moor (TASK-138)

> status: diagnosis and fix 2026-11-07. Tags: V verified live (how stated), S read in code, U unverified. Evidence of the first run: `docs/tester-live.md` (D8), `docs/tester-realtime.md`.

## What was reproduced (V)
Same scenario as the first run: image from the public fork tag (`didle-gate2-lab3`) built with `deploy/d2d-container/context.sh`, wizard path, Sorceress, goals `campaign:normal_act1`, realtime, game folder read-only with the four required MPQs, throwaway character, compose project `d2tester138-20261107` on a high loopback port. No lab verbs, no memory writes: writes after start 0.

Timeline of the unfixed bot (bot `minute` and `recovery:` lines):
- Second 0: `recovery: start (gold 0 < kit 500)`. A new hero has 0 gold, a town portal scroll, 4 minor healing potions and a tome; `KitShortfall` (190) is above 0, so recovery starts in town and stays "Recovery: farming gold".
- Minute 3 to 5: the 4 potions are drunk at 16 to 21 of 40 life (clvl 1 against Quill Rats and Fallen), no gold to buy more, `recovery: kit gate on`, `poverty mode: on`.
- Gold income: 5 gold after 17 kills, 28 after 11 min, +3 net after 22 min (a mana potion bought). About 100 gold per hour against a bar of 500 (target 900).
- XP: clvl 2 at minute 9, clvl 3 at minute 24 (xp 1605), 66 kills. After that the field is depleted: 45 kills in the first 25 min of the second run, then a straggler every 1 to 5 minutes (48 kills, xp 1083 at minute 40 of that run). The first realtime hour ended at 128 kills, clvl 3.

## Causes
1. Recovery never ends (V). Its exit needs gold 900 (not reachable at 100 gold per hour at clvl 1 to 4) or the 45 minute cap; when the cap ends it, `KitLow` (no potions) bypasses the 15 minute cooldown and starts it again at once (S, `internal/resource/recovery.go`). While it runs it replaces the campaign's quest step (S, `recoveryStep`), so the hero could not take a campaign step (Den of Evil, clvl 5 for a caster) even if its level allowed it.
2. The field is depleted and nothing notices (V). In one game Blood Moor holds a few dozen monsters. After that the hero finds one every few minutes ("clear killed nothing; moving on" every minute). The kill drought watchdog (`bot.droughtMin`, 8 minutes without ANY kill) never fires because each straggler resets it. A fresh game refills the field. This, not the gold, is what cost the XP: the first realtime hour spent about 35 of its 60 minutes in an empty field.
3. Not causes (V/S): not the merc (`merc: gold farm on need=710` at clvl 1 is only a log line, no revive was charged; no merc exists), not a realtime pacing bug (kill timing, `stallSec` 0; the lockstep short drill of `docs/probes/shortdrills.md` reached clvl 4 at about 49 virtual minutes, the same pace), not the area choice (the XP pick picks Blood Moor at clvl 1, `xp/kill est 22`).
4. The pace itself (V): a clvl 1 Sorceress is mana limited (35 mana, about 0.2 mana per second regeneration, `combat resting before the pack ... waitSec 21`): 3 to 7 kills per minute while monsters last. "clvl 5 to 8 in the first hour" is not reachable in one empty-prone Blood Moor game; clvl 5 needs 7500 XP.

## Fix
- `internal/resource/recovery.go`, knobs `resource.recoveryStallMin` (20), `resource.recoveryStallGold` (100), `resource.recoveryStallHoldMin` (60): a recovery that gained under 100 gold after 20 minutes ends with the reason `stalled`, and the low-kit bypass does not restart it for 60 minutes (a death streak still does). The planner then runs its normal step (`level.N`, quest steps) instead of the gold loop. Class free: no class or skill literal.
- `internal/bot/drought.go`, knobs `bot.droughtSlowMin` (10), `bot.droughtSlowKills` (3), `bot.droughtSlowMaxLevel` (10): after 10 minutes in one field area at clvl below 10 with fewer than 3 kills, the bot starts a fresh game (stage 2 of the drought ladder, so `freshAfterRecoverySec` and a missing cycler still apply). Off with 0.
- Tests: `TestRecoveryStallsAtLowIncome` (resource; two tunings), `TestSlowFieldStartsAFreshGame` and `TestSlowFieldQuietCases` (bot; two classes). Both fail on the code before.

## Drill after the fix
Run 2 (stall rule only) and run 3 (both rules), fresh volume each, container left running 40 to 45 minutes (not a full hour), writes after start 0.
- Run 2 (V): `recovery: end (gold 11, reason stalled)` at 20 min 20 s; recovery did not restart; the planner went on with `level.5`. The field was then empty: 48 kills and xp 1083 (clvl 2) at minute 40, one kill per 1 to 5 minutes. The old kill drought did not fire (a straggler resets it), which led to the slow-field rule.
- Run 3 (V, both rules): `recovery: end (gold 8, reason stalled)` at minute 25; the old kill drought stage 1 then left Blood Moor (`stage 1 ... minutes 8`), the hero visited Cave Level 1 (area 9), shopped and came back; 61 kills, xp 1437 (clvl 2) at minute 38. The slow-field rule never fired in this run (U live): there were always 3 or more kills in a 10 minute window. It is proven offline only.
- XP per hour is NOT shown improved (V): all three runs sit at 25 to 35 xp per minute early (xp 1605 at minute 24 unfixed, 1083 at 40 with the empty field, 1437 at 38). The hero stays at clvl 2 to 3 after 40 minutes; clvl 5 to 8 in the first hour was not reached and is U for the fixes.

## Not covered
Reaching the Den of Evil (clvl 5, 7500 XP) inside one realtime hour was not shown (U): the pace above makes it an hour-two event. Gold income at clvl 1 to 4 is unchanged; the kit stays unaffordable until loot sells or the level rises.
