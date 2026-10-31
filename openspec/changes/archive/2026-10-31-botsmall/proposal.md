## Why

Three small bot-code defects from the backlog: the minute line's restarts counter misses game recycles on the d2d backend, d2.Kind() labels champions as unique, and the two fresh-game resets duplicate the same per-game forgetting.

## What Changes

- Minute line restarts counts every bot-started game (newGameRun, Recycle) as well as menu-observed entries.
- MonsterInfo.Kind and IsUnique test the champion bit before the boss bit (champion reads 0x0c).
- One forgetGame helper for syncGames and freshGameToLevel.
- Shared bot loop code: needs a G1/G2 check. See docs/bot-small.md.
