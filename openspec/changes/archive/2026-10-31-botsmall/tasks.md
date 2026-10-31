# Tasks

```yaml
change: botsmall
status: archived
agent: botsmall
branch: agent/botsmall
worktree: ~/code/d2lod-wt/botsmall
owns:
  - internal/bot/observe.go
  - internal/bot/farm.go
  - internal/bot/goal.go
  - internal/bot/loop.go
  - internal/bot/task.go
  - internal/bot/botsmall_test.go
  - internal/d2/units.go
  - internal/d2/monflags.go
  - internal/d2/monflags_test.go
  - internal/combat/target.go
  - docs/bot-small.md
  - openspec/changes/botsmall/**
exclusive: []
gates:
  - "go test of internal/bot, internal/d2, internal/combat passes; whole-repo vet and short tests pass on the worker host"
merge: c5b4ecc
result: "offline only, not verified live; item 4 skipped (see docs/bot-small.md)"
```

## 1. Items

- [x] 1.1 restarts counter counts every recycle (table test)
- [x] 1.2 champion classification (table test on real flag bytes)
- [x] 1.3 one forgetGame helper (test for the observer path)
- [ ] 1.4 potionsDrunk vs belt reads: skipped, needs live data
