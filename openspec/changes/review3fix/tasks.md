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
  - internal/bot/areaskip_test.go
  - docs/plan/data/progression-normal.json
  - docs/plan/data/runs.json
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "C1-C4 fixed offline, unit tests only, not verified live. C4 also synced docs runs.json and the Cave 2, Crypt, Mausoleum steps (docs copy was stale)."
```

- [x] C1 route bans ignore nav-fault skips of non-target hops
- [x] C2 targetFails decay
- [x] C3 failed ToTown counts in recKitTrips
- [x] C4 plan flags and TestMatchesDocs
