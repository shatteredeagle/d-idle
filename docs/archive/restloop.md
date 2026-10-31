# Sorceress rest loop on i4 (diagnosis from the log, no game)

Log: `integrate2/20261029-000335-d2bot` (the worker host), Booti clvl 12 Sorceress, 00:12-00:16. Tags: V = read in that log or the code, U = inferred.

| Symptom | Cause | Status |
|---|---|---|
| Clears Blood Moor under step `level.14` at level 12 | not the leveling planner: at 00:12:28 `starving: no potions and no gold; farming gold in an easy area` then `planner: starvation detour` area 2. The task label is just the active quest step. Gold 0 because the 00:07 portal/town trip failed (`no town portal available`) and `poverty rebuy` spent 0. | V |
| `combat resting before the pack`, mana 9-14, target 36, ~40 s, then `rest timed out` | target = 60% floor (`RestStartManaPct`) of max mana 60, applied even to ONE trash monster; regen measured 13 mana/25 s and 15/31 s = about 0.5/s, i.e. 115-125 s to full (U `manaFullSecs` = 100 is 20% optimistic; docs/resource.md measured 130 s earlier). No gold for mana potions, so every rat costs a rest. | V (log numbers) |
| `clear killed nothing; moving on`, remaining 0, hostileNear 3-4 | `bot.hostileNear` counted units the Fighter had already given up on (`combat ignoring target ... approaches`, memory `IgnoreMemorySec` x strikes); `Clear` skips those (`ignoredNow`) and `Remaining` only counts this fight's skipped/avoided, so it returns Killed 0 in ~50 ms. Friendly-vs-hostile mismatch is not the cause in this build (old pre-IsFriendly code, no summons exist for a Sorc at clvl 12 without Hydra/Golem). | V code path, U which units |
| `combat kill ... xp:0` | partly real: clvl 12 vs Fallen/Quill Rat/Zombie gives 1-2 xp (level penalty, U exact values). Partly a bug: `noteKills` took `pendingXP/len(died)` and zeroed the pool, losing the remainder; and exp can arrive a tick after the death is seen, so it lands on the next kill. | V remainder, U lag |

## Fixed
- `resource.PreEngage`: a pack of at most 2 with danger 0 only needs its cast budget (`n * CastsPerMonster * cost`), not the 60% floor. (`internal/resource/preengage.go`, test `TestPreEngageTrashPackNeedsOnlyCastBudget`; `TestRestsBeforeAFarPack` now uses a pack of 3.)
- `bot.hostileNear` skips units where the Fighter's optional `IsIgnored(id, dist)` is true (`cmd/d2bot` adapter forwards `combat.Fighter.IsIgnored`). Test `TestHostileNearSkipsIgnored`.
- `combat.noteKills` keeps the XP remainder for the next kill. Test `TestNoteKillsKeepsXPRemainder`.

## Not fixed (listed)
- The kill XP attribution lag (exp read a tick after the corpse) needs a live frame trace: U.
- `manaFullSecs` should be ~125 (V two samples in this log, 130 earlier); left at 100 because it feeds town-trip starvation tests.
- Why `no town portal available` at 00:07 (no TP tome, gold 0) and why the repair at 00:09 failed ("durability 101% unchanged") belong to town/chores, not looked at.
- Live check: with a Sorc and 0 gold, confirm the loop of rest/Clear-nothing is gone and rats are fought on 9-14 mana (staff fallback).
