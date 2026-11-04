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
  - openspec/changes/tomefix/**
exclusive: []
gates:
  - "go test -p 1 for each touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: ""
live: "not verified live"
```

- [ ] audit the false-empty paths
- [ ] regression tests with fakes
- [ ] fix only proven gaps
- [ ] live-check recipe
