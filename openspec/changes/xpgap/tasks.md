# Tasks

```yaml
change: xpgap
status: in-progress
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
merge:
result:
```

## 1. Work

- [x] 1.1 diagnose from the soakfull logs
- [x] 1.2 knobs recoveryXPEarly and povertySellMin with offline tests
- [ ] 1.3 A/B on hidden d2d
- [ ] 1.4 docs/d2d-xp-gap.md
