# Tasks

```yaml
change: xpgap
status: archived
agent: xpgap
branch: agent/xpgap
worktree: ~/code/d2lod-wt/xpgap
owns:
  - docs/d2d-xp-gap.md
  - docs/knobs.md
  - internal/bot/goal.go
  - internal/env/tuning_resource.go
  - internal/resource/tuning.go
  - internal/env/tuning_town.go
  - internal/town/chores.go
  - internal/town/shop.go
  - internal/town/town_test.go
  - cmd/d2bot/resource_test.go
  - internal/bot/recoveryxp.go
  - internal/bot/recoveryxp_test.go
  - internal/bot/povertysell_test.go
  - tools/xpgap/**
  - openspec/changes/xpgap/**
exclusive:
  - soak-slot:3
gates:
  - "go test -p 1 for the touched packages and whole-repo vet/test on the worker host"
  - "before/after on one binary, 4 seeds x 2 virtual hours per arm, interleaved"
merge: f62a9f5
result: "verified live on hidden d2d: diagnosis from 12 soak logs; 3 knobs (default off) and recoveryOn=0 tested, none beats noise (4 runs per arm, SE 7-9 %), recoveryOn=0 is -15 %; vet and short tests clean on the worker; not run on Game.exe"
```

## 1. Work

- [x] 1.1 diagnose from the soakfull logs
- [x] 1.2 knobs recoveryXPEarly and povertySellMin with offline tests
- [x] 1.3 A/B on hidden d2d (off, on, sell, tp, norec arms; 24 runs)
- [x] 1.4 docs/d2d-xp-gap.md
