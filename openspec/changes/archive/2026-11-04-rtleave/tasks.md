# Tasks

```yaml
change: rtleave
status: archived
agent: rtleave
branch: agent/rtleave
worktree: ~/code/d2lod-wt/rtleave
skip_specs: true
owns:
  - docs/rtleave.md
  - internal/drill/retreat_leave_test.go
  - openspec/changes/rtleave/**
  - cmd/d2drill/leave_lab.go
  - cmd/d2drill/leave_lab_test.go
  - cmd/d2drill/retreat_lab.go
  - cmd/d2drill/live.go
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "offline; live d2-live run pending"
live: "not verified live"
merge: 902f087
```

- [x] work done
