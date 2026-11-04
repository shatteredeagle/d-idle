# Tasks

```yaml
change: tomefix
status: in-progress
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
result: "audit and 14 regression tests pass offline (town ran on the worker host); no bot gap found, no code change; producer paths 9/10 and fork qty stay U"
live: "not verified live"
```

- [x] audit the false-empty paths
- [x] regression tests with fakes
- [x] fix only proven gaps (none proven)
- [x] live-check recipe
