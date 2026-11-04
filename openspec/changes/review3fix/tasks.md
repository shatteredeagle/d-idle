# Tasks

```yaml
change: review3fix
status: in-progress
agent: review3fix
branch: agent/review3fix
worktree: ~/code/d2lod-wt/review3fix
skip_specs: true
owns:
  - openspec/changes/review3fix/**
  - internal/bot/goal.go
  - internal/bot/geargate.go
  - internal/bot/areaskip.go
  - internal/bot/task.go
  - internal/bot/runner.go
  - internal/bot/area_ban_test.go
  - internal/bot/areaskip_test.go
  - internal/plandata/plandata_test.go
  - docs/plan/data/progression-normal.json
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: ""
```

- [x] C1 route bans ignore nav-fault skips of non-target hops
- [ ] C2 targetFails decay
- [ ] C3 failed ToTown counts in recKitTrips
- [ ] C4 plan flags and TestMatchesDocs
