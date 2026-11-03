# Tasks

```yaml
change: gate1fix
status: in-progress
agent: gate1fix
branch: agent/gate1fix
worktree: ~/code/d2lod-wt/gate1fix
skip_specs: true
owns:
  - openspec/changes/gate1fix/**
  - internal/bot/task.go
  - internal/bot/task_test.go
  - internal/bot/areaskip.go
  - internal/bot/runner.go
  - internal/bot/fakes_test.go
  - internal/bot/waypoints_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/bot passes; changecheck 0 warnings"
result: "Offline only, not verified live. R1 and R2 fixed with table tests at two character levels."
```

## 1. Work
- [x] R1 longestWalk honours failed, short and disabled waypoint jumps
- [x] R2 failed walk skips the target with a doubling cool-down
- [x] Diagnose the seed 17 and seed 19 windows (in the report)
