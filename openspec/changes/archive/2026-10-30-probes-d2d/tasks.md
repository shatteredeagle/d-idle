# Tasks

```yaml
change: probes-d2d
status: archived
agent: probes-d2d
branch: agent/probes-d2d
worktree: ~/code/d2lod-wt/probes-d2d
owns:
  - cmd/d2dspawn/**
  - cmd/d2dcasts/**
  - docs/d2d-tick-rate.md
  - docs/d2d-spawns.md
  - docs/d2d-casts.md
  - docs/probes/**
exclusive: []  # the lab lock was used only through flock -w (queued behind D2DLOOP), never held outside a run
gates:
  - "every claim tagged V-d2d or U"
merge: 74d6bbe
result: 'd2d side of the first 3 probes: tick rate 25 ticks per game second, 1 tick = 1 original frame (V-d2d); spawn counts 5 areas x 20 seeds (docs/probes/d2d-spawns.csv; original-side columns empty); casts per kill vs Fallen Charged Bolt 2.59/1.18/0.97 at skill 1/4/8; XP log line already names the monster'
```

## 1. Work

- [x] 1.1 T1 tick rate: code read plus one measured check, docs/d2d-tick-rate.md
- [x] 1.2 T2 spawns: 5 areas x 20 seeds, CSV plus table
- [x] 1.3 T3 casts per kill vs Fallen, Charged Bolt Sorceress
- [x] 1.4 XP log line carries the monster name (already true in internal/combat, see docs/d2d-casts.md)
