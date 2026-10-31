## Why

The d2d bot hangs deterministically under a lockstep clock (soakfull seed 11 at virtual minute 109, runab seed 42 walk-only at about 90 s): the combat loop spins on a crowd step-away whose walk fails at once with `nav: no path: empty path`. Under lockstep, game time only moves on sleeps and steps, so a loop whose iterations neither sleep nor step never advances time and never ends.

## What Changes

- `Fighter.move` (the one place combat walks through the Mover): a failed walk now costs one poll tick.
- `canRetreat` is false while the fighter is pinned, so the crowd step-away is not offered when it cannot work.
- Fight-loop no-progress watchdog: 40 iterations without the clock moving force one poll sleep and log `combat no progress, holding` once per fight.
- Tests: offline repro (crowd, failed walks, bounded), pinned retreat, failed move advances time, watchdog with a frozen clock.
- Loop audit (town, loot, bot, nav): all other `for {` loops sleep, are bounded by tries or hops, or end on a deadline; no change needed.
