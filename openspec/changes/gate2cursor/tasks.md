# Tasks

```yaml
change: gate2cursor
status: in-progress
agent: gate2cursor
branch: agent/gate2cursor
worktree: ~/code/d2lod-wt/gate2cursor
skip_specs: true
owns:
  - docs/gate2cursor.md
  - openspec/changes/gate2cursor/**
  - internal/town/hand/drop.go
  - internal/loot/pickup.go
  - internal/loot/loot_test.go
  - internal/town/sim/input.go
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "Offline root cause (log + code) for the cursor fault, fix + 2 fake tests (pass on the worker host); the no-area block explained, no policy change. Not verified live."
live: "not verified live"
```

- [x] root cause A (cursor) and B (no leveling area) from the log and code
- [x] fake regression tests (loot)
- [x] minimal fix: drop a held item no pack cell takes
- [x] docs/gate2cursor.md
