# Tasks

```yaml
change: economy
status: archived
agent: economy
branch: agent/economy
worktree: ~/code/d2lod-wt/economy
owns:
  - docs/d2d-economy.md
  - tools/d2dloop/**
  - internal/d2denv/env.go
  - internal/town/chores.go
  - internal/bot/goal.go
  - internal/bot/povertysell_test.go
exclusive:
  - d2d-lab-binary
gates:
  - "before and after XP/h, kills/h, gold/h and recovery share on the same four seeds"
merge: 6e0bb1e
result: "d2d economy: gold pickup works (0.3-0.4 gold/kill, matches original); fixed scroll-burn loop (povertySellWorth: 6 junk items with TP scroll, 4 without; shared with Game.exe: needs a G1/G2 check), d2d link-wait (LateClock.Poll 2 ticks + 10 ms wall: link waits ate 15-25% of virtual time, 7-10 failsafe resets per run -> 0); 4 seeds x 2 virtual h: XP/h 3013 -> 3869 (+28%), kills/h 132 -> 167; gate NOT met (baseline 6600); remaining: 0-kill recovery minutes, repair fails sometimes (diagnostic log added)"
```

## 1. Work

- [x] 1.1 diagnose sell, repair, gold and time sinks on d2d
- [x] 1.2 fix the root causes found, with tests
- [x] 1.3 before and after on seeds 11 to 14
