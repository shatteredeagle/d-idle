# Tasks

```yaml
change: rtleave
status: in-progress
agent: rtleave
branch: agent/rtleave
worktree: ~/code/d2lod-wt/rtleave
skip_specs: true
owns:
  - docs/rtleave.md
  - internal/drill/retreat_leave_test.go
  - openspec/changes/rtleave/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: ""
live: "not verified live"
```

- [ ] work done
