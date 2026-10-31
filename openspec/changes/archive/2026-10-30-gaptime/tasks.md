# Tasks

```yaml
change: gaptime
status: archived
agent: gaptime
branch: agent/gaptime
worktree: ~/code/d2lod-wt/gaptime
owns:
  - docs/d2d-casts-real.md
  - docs/probes/d2d-casts-real.csv
  - docs/research/per-kill-time-split.md
  - cmd/d2dcasts2/**
exclusive:
  - d2d-lab-lock
gates:
  - "every number tagged V/S/U"
merge: c137595
result: 'real d2d casts per Fallen kill 1.16/0.73/0.61 (castGapMs does not change it); seconds per kill early char Blood Moor+Cold Plains: original 9.1 vs d2d 23.6; fight+approach 5.3 vs 12.2, search/stall 0.3 vs 5.5, town/recovery 3.0 vs 5.2; bot-side gap'
```

## 1. Work

- [x] 1.1 T1 real casts per kill (mana / cost), castGapMs 450 and 650
- [x] 1.2 T2 per-kill time split on both backends
- [x] 1.3 T3 explanation ranking and next experiment
