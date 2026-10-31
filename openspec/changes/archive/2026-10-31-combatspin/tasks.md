# Tasks

```yaml
change: combatspin
status: archived
agent: combatspin
branch: agent/combatspin
worktree: ~/code/d2lod-wt/combatspin
owns:
  - internal/combat/fight.go
  - internal/combat/combatspin_test.go
  - docs/combatspin.md
  - openspec/changes/combatspin/**
exclusive: []
gates:
  - "go test -p 1 ./internal/combat passes; the offline repro does not trip the watchdog"
  - "whole-repo go vet and go test -short pass; go test -race ./internal/combat"
result: "NOT verified live (seed 2 replay skipped: the fix changes flee decisions near mouths so the run diverges anyway); offline repro fails before the fix and passes after; vet, short tests and race for internal/combat pass on Orion; Game.exe-affecting (shared combat code)"
merge: 7d8fe06
```

## 1. Fix

- [x] 1.1 find the path (abortFight flee to a cave-mouth point, move skipped without a tick)
- [x] 1.2 fix in move and valid, offline tests
- [x] 1.3 whole-repo checks on the worker host
- [ ] 1.4 live verify seed 2
