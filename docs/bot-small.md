# Bot small fixes (change botsmall)

> status: notes of one small change card; offline only, nothing here is verified live.

## Minute line restarts
- Old: `max(0, newGames-1-plannedGames)`; newGames moves only when the menu observer sees a menu then a game. The d2d backend restarts the process without a menu poll, so it read 1 of 17 recycles.
- Now: `restartCount(newGames, planned, recycles)` = max of the menu-based count and `ob.recycles`, which `newGameRun` (farm, difficulty switch, fresh game, failsafe rung 3) and the betweenGoals recycle bump on success. Save points stay planned (oogRun, not counted). Crash re-entries on d2d are still not counted (nothing sees them).
- Needs a G1/G2 check: log-only change on the Game.exe path (max of two counts, never lower than before).

## Monster kind
- Bits VERIFIED LIVE (original-game census, docs/orig-spawns.md): 0x04 champion (reads 0x0c), 0x08 rare/unique (0x0a superunique), 0x10 follower, ordinary monsters read 0x00 (bit 0x02 never set), so Kind stays Unknown for 0x00.
- Kind() tests champion before boss; IsUnique() excludes the champion bit.
- Behaviour changes: boss.go / bossrun adapter (cmd/d2bot, cmd/bossrun) no longer give champions a UniqueNo or boss name; combat/resource.go and affix.go Kind users now see Champion instead of Unique (resource.go only gates on Normal, no change). target.go and observe.go (boss flag, killerBoss) deliberately keep champion = boss, so fight and autopsy behaviour is unchanged. Needs a G1/G2 check only for the boss-run view.
- cmd/origcasts (not owned) still carries a comment that Kind tests the boss bit first; it can be dropped.

## Fresh-game reset
- `forgetGame` (goal.go) is the single reset used by syncGames and freshGameToLevel; freshGameToLevel now also clears waitNote (harmless). Needs a G1/G2 check (shared loop).

## Skipped
- potionsDrunk vs belt reads: the counter is event-based (EvPotion), the belt read comes from snapshots; comparing them needs live belt data, no cheap offline test.
