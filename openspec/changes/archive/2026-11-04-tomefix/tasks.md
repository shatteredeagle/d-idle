# Tasks

```yaml
change: tomefix
status: archived
agent: tomefix
branch: agent/tomefix
worktree: ~/code/d2lod-wt/tomefix
skip_specs: true
owns:
  - docs/tomefix.md
  - internal/town/tomefix_test.go
  - internal/combat/tomefix_test.go
  - internal/d2denv/tomefix_test.go
  - openspec/changes/tomefix/**
exclusive: []
gates:
  - "go test -p 1 for each touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "offline audit: no path found; stays U: stock items verb and dump items failure, fork qty dropout"
live: "not verified live"
merge: 024c759
```

- [x] audit the false-empty paths
- [x] regression tests with fakes
- [x] fix only proven gaps (none proven)
- [x] live-check recipe
