# Tasks

```yaml
change: m1-stashcap
status: deferred
agent: m1-stashcap
branch: agent/m1-stashcap
worktree: ~/code/d2lod-wt/m1-stashcap
owns:
  - internal/town/stash.go
  - internal/town/stashcap_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/town passes including TestStashGoldCapDeposit (partial deposit to the cap, one log line, no retry on the next visit while full)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/basics-audit.md E4 and G10 and internal/town/stash.go; failing test first
- [ ] 1.2 Partial deposit and the once-only log; cap constant tagged with its source
- [ ] 1.3 changecheck, leakcheck, validate; report the row that flips
