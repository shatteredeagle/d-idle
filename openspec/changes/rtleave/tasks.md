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
  - cmd/d2drill/leave_lab.go
  - cmd/d2drill/leave_lab_test.go
  - cmd/d2drill/retreat_lab.go
  - cmd/d2drill/live.go
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "Fighter.Leave wired in d2drill (Save and Exit, rejoin), offline tests pass; not verified live"
live: "not verified live"
```

- [x] work done
