# Tasks

```yaml
change: d2dwire
status: archived
agent: d2dwire
branch: agent/d2dwire
worktree: ~/code/d2lod-wt/d2dwire
owns:
  - cmd/d2bot/d2d*.go
  - cmd/d2bot/mercwire.go
  - internal/d2dcombat/rig.go
  - internal/d2dsrc/source.go
  - docs/d2d-wire.md
exclusive:
  - d2d-control-socket
gates:
  - "go test ./cmd/d2bot -run D2D passes (whole-stack and hookup tests)"
  - "a stranger socket is never driven or quit (TestManagedRefusesStrangerSocket)"
  - "live whole-stack run on d2d: U, waits for the lab"
merge: 648a333
result: "whole stack wired on d2d (combat, town, task deps, merc seam, claim, heartbeat, socket ownership); live: never past clvl 1 (0 gold, 1-2 deaths), loot not wired, repair/stash/portal geometry open, lockstep progress clock open"
```

## 1. Work

- [x] 1.1 Build the real stack over d2d seams and test the wiring offline
- [x] 1.2 Socket ownership safety and tests
- [x] 1.3 Fix the d2dsrc warnOnce race found by the whole-stack test
- [x] 1.4 docs/d2d-wire.md
- [ ] 1.5 Live whole-stack run on d2d (needs lab, U)
