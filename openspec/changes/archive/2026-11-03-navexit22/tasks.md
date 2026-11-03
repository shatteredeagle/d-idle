# Tasks

```yaml
change: navexit22
status: archived
agent: navexit22
branch: agent/navexit22
worktree: ~/code/d2lod-wt/navexit22
skip_specs: true
owns:
  - internal/nav/exit.go
  - internal/nav/seed17_test.go
  - internal/d2dworld/testdata/seed17/**
  - docs/navexit22.md
  - openspec/changes/navexit22/**
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dworld ./internal/nav ./internal/bot passes"
  - "Gate 1 seed 17 re-run: no `no exit to that level: level 1 from 2` (live, not done)"
result: "Offline only, not verified live. Root cause found by recording seed 17 from a real d2d and replaying it (TestSeed17BloodMoorHasTownGate fails before, passes after): borderGates compared frames wrongly."
credit:
  author: owner
  agents: [Claude Code]
merge: a1401fd
```

## 1. Work

- [x] 1.1 Record seed 17 replies from a throwaway d2d slot
- [x] 1.2 Replay test that fails before the fix
- [x] 1.3 Frame fix in borderGates
- [ ] 1.4 Gate 1 seed 17 re-run (live)
