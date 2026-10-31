## Why

Live (RUNFIX): a fresh Sorceress on d2d livelocks at clvl 3 with 'leveling: everything worth farming is cleared, starting a fresh game' hundreds of times a minute. Cause: the task path (taskActs.LevelTo) never called syncGames, so loop.cleared kept the previous game's areas after NewGame and freshGameToLevel fired again at once.

## What Changes

- LevelTo and taskEnv.World call syncGames first (cleared set, sweep kills, step area reset on a new game).
- test TestFreshGameDoesNotRetriggerBeforeANewClear (failed on master with 4 fresh games, passes now).
- Game.exe-affecting (shared internal/bot): needs a G1/G2 check; the original path has the same bug if the task path runs there.
