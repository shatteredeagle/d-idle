# Tasks

```yaml
change: d2dloot
status: archived
agent: d2dloot
branch: agent/d2dloot
worktree: ~/code/d2lod-wt/d2dloot
owns:
  - internal/d2dloot/**
  - cmd/d2bot/d2dloot.go
  - cmd/d2bot/d2dloot_test.go
  - docs/d2d-loot.md
exclusive:
  - d2d-lab-binary
gates:
  - "gold and items picked up on the private fork with autoloot off, numbers in docs/d2d-loot.md"
merge: 476dbfd
result: "Loot seam wired on d2d: ground verb adapter, pickup by cmd pickup, 8 live pickups 0 failures; whole bot never reached a pickup (clvl 1 Den problem); quality labels beyond normal unverified"
```

## 1. Work

- [x] 1.1 d2dloot adapter and Loot seam, offline tests
- [x] 1.2 Live run on the private fork, numbers in docs/d2d-loot.md
