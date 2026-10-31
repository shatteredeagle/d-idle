# Tasks

```yaml
change: d2dlevel
status: archived
agent: D2DLEVEL
branch: agent/d2dlevel
worktree: ~/code/d2lod-wt/d2dlevel
owns:
  - internal/bot/**
  - cmd/d2bot/d2dwire.go
  - cmd/d2bot/d2dbackend.go
  - docs/earlygame.md
  - docs/d2d-pace.md
exclusive:
  - d2d-lab-lock
gates:
  - "no change to Game.exe behaviour except a shared bug with a test"
merge: b2713c9
result: "Den floor no longer waived far below the floor (shared bug, tested); root cause B (d2d returns the whole map so sweep marks areas cleared at once, clvl stays 1) handed to D2DROAM; live check not done"
```

## 1. Work

- [x] 1.1 Reproduce clvl 1 Den entry offline (test)
- [x] 1.2 Fix the floor yield
- [ ] 1.3 Live check on d2d.lab2
