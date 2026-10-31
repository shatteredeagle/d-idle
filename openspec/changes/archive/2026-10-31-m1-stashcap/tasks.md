# Tasks

```yaml
change: m1-stashcap
status: archived
agent: m1-stashcap
branch: agent/m1-stashcap
worktree: ~/code/d2lod-wt/m1-stashcap
owns:
  - internal/town/stash.go
  - internal/town/town.go
  - internal/town/basicsfix.go
  - internal/town/sim/**
  - internal/town/stashcap_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/town passes including TestStashGoldCapDeposit (partial deposit to the cap, one log line, no retry on the next visit while full)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: 4d26ed1
result: stashcap implemented offline (not verified live; cap constant 2,500,000 UNVERIFIED until m1-liveproof). d2d-only: no (shared town code, sim-tested). No knob: the cap is a game fact and the old behaviour was a failure loop.
```

## 1. Work

- [x] 1.1 Read docs/basics-audit.md E4 and G10 and internal/town/stash.go; failing test first
- [x] 1.2 Partial deposit and the once-only log; cap constant tagged with its source
- [x] 1.3 changecheck, leakcheck, validate; report the row that flips
