# Tasks

```yaml
change: d2dpace
status: archived
agent: d2dpace
branch: agent/d2dpace
worktree: ~/code/d2lod-wt/d2dpace
owns:
  - internal/d2denv/**
  - cmd/d2bot/d2dwire.go
  - cmd/d2bot/d2dbackend.go
  - docs/d2d-pace.md
  - docs/d2d-wire.md
exclusive:
  - d2d-lab-lock
gates: []
merge: a576af7
result: "kills/potions to the summary via lateEmit, Pacer virtual clock carried across restarts and failed steps wait in wall time, d2dTown opens the portal itself; live: campaign still clvl 1 in 4 runs; blockers: no gold, Den at clvl 1, inventory flag not reported, lockstep only 0.5 virtual s per wall s"
```

## 1. Work

- [x] 1.1 Audit time sources on d2d
- [x] 1.2 Fix minute-line counters
- [x] 1.3 Live progression sessions and findings
