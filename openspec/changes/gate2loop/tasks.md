# Tasks

```yaml
change: gate2loop
status: in-progress
agent: gate2loop
branch: agent/gate2loop
worktree: ~/code/d2lod-wt/gate2loop
skip_specs: true
owns:
  - docs/gate2loop.md
  - internal/bot/savepoint.go
  - internal/bot/savepoint_test.go
  - internal/bot/goal.go
  - openspec/changes/gate2loop/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "save point now runs after the town chores and only when XP was earned since the last entry; fake tests added; not verified live (needs a Gate 2 seed 13 rerun)"
live: "not verified live"
```

- [x] work done
