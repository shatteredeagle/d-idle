# Tasks

```yaml
change: d2d-loop-hours
status: archived
agent: d2d-loop-hours
branch: agent/d2d-loop-hours
worktree: ~/code/d2lod-wt/d2d-loop-hours
owns:
  - docs/d2d-loop-hours.md
  - docs/d2d-loop.md
exclusive:
  - d2d-control-socket
  - d2d-lab-lock
gates:
  - "at least 4 virtual hours without a bot stall longer than 10 minutes, or the stall causes are listed"
  - "every claim tagged V-d2d"
merge: 56502ea
result: "D2DLOOP: lab5 deterministic for scripted walk/fight (3/3 identical hashes), full bot loop not deterministic; fresh Sorceress seed 5 no cheats 8 virtual h: clvl 6, 762 kills, 1 death; replicated 4 seeds x 2h: XP/h 3013 vs 6600 baseline (-54%), kills/h 132 vs 500, deaths 0, verdict INCONCLUSIVE, 20% gate NOT met; main gap poverty (recovery 63% of time, ~20 gold/h); fixes: loot seam, cursor item, belt fill, stat points, panels, Den/cave-mouth shuttle, stale roam coverage, chicken restart halt, BackToField, poverty sell trips; not verified live on Game.exe"
```

## 1. Work

- [x] 1.1 run the full loop for at least 4 virtual hours per seed, 3 seeds (partly: 4 seeds x 2 h and 3 single runs of 6 to 8 h on seed 5)
- [x] 1.2 record crashes, stalls, loop gaps (loot, merc, Act 2 are known stubs)
- [x] 1.3 report gaps as follow-up cards
