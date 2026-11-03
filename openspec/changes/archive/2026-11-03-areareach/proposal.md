## Why

Gate 1 run 0628 fell to 7060 XP/h because the planner refused Dark Wood, Black Marsh and Cave Level 2 (28 times, "no activated waypoint on the way") whenever the bot stood in a field or dungeon, and the fallback was Cold Plains at 3.4 XP per kill (docs/research/xphshort.md).

## What Changes

- bot.unreachable: the walk between activated waypoints (town counts) may be up to 5 hops, from wherever the character stands. Before, it had to stand at a waypoint or in town.
- Plan.Pick: when the area the plan prefers is excluded, the fallback is the eligible step with the best XP per kill (areaXP) instead of the last in progression order.
- Belt, shop and walk-home code is untouched.

## Impact

internal/bot/task.go, plan.go, goal.go, recoveryxp.go and their tests. No spec delta.
