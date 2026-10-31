# Tasks

```yaml
change: d2droam
status: archived
agent: d2droam
branch: agent/d2droam
worktree: ~/code/d2lod-wt/d2droam
owns:
  - internal/nav/**
  - internal/d2dworld/**
  - docs/nav.md
  - docs/d2d-pace.md
exclusive:
  - d2d-lab-lock
gates:
  - "Game.exe explore path unchanged (tests for both sources)"
  - "live d2d numbers reported as V only if measured"
merge: 80f880a
result: "FullyKnown capability and roam sweep on d2d: coverage points 7x7 per block; live Blood Moor 1764 points (too many for the sweep timeout); next: sparser spacing/cap; sweeps log coverage N/M"
```

- [x] nav roam + FullyKnown + tests
- [x] bot log line and kill count (minimal hook in internal/bot/goal.go)
- [x] docs/nav.md note, docs/d2d-pace.md section 'roam'
- [x] live run on the d2d fork
