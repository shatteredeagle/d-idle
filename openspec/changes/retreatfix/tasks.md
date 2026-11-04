# Tasks

```yaml
change: retreatfix
status: in-progress
agent: retreatfix
branch: agent/retreatfix
worktree: ~/code/d2lod-wt/retreatfix
skip_specs: true
owns:
  - docs/retreatfix.md
  - internal/drill/retreat.go
  - internal/drill/retreat_test.go
  - openspec/changes/retreatfix/**
exclusive: []
gates:
  - "go test -p 1 ./internal/drill ./cmd/d2drill passes; changecheck 0 warnings; leakcheck 0 hits"
result: ""
live: "not verified live"
```

- [ ] work done
