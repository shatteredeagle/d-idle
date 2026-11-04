# Tasks

```yaml
change: retreatdrill
status: in-progress
agent: retreatdrill
branch: agent/retreatdrill
worktree: ~/code/d2lod-wt/retreatdrill
skip_specs: true
owns:
  - docs/retreatdrill.md
  - internal/drill/retreat_test.go
  - openspec/changes/retreatdrill/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: ""
live: "not verified live"
```

- [ ] work done
