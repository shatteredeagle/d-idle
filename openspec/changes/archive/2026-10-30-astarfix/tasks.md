# Tasks

```yaml
change: astarfix
status: archived
agent: astarfix
branch: agent/astarfix
worktree: ~/code/d2lod-wt/astarfix
owns:
  - internal/nav/astar.go
  - internal/nav/astar_race_test.go
  - internal/nav/nav.go
  - internal/d2denv/proc.go
exclusive: []
gates:
  - "go test -race ./internal/nav passes"
merge: d3f98b1
result: 'nav Planner data race fixed: guard goroutine and acting goroutine shared the plan path; mutex in Planner.Plan and Navigator.planHard; race test fails on old code; benchmark unchanged (noise); Proc.Start waits for a running process instead of failing with already running x5 (needs live restart check)'
```

## 1. Work

- [x] 1.1 race test (fails on master under -race)
- [x] 1.2 mutex in Planner.Plan and Navigator.planHard
- [x] 1.3 d2d already running: Start waits for the running process
