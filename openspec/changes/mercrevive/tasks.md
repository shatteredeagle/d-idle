# Tasks

```yaml
change: mercrevive
status: in-progress
agent: mercrevive
branch: agent/mercrevive
worktree: ~/code/d2lod-wt/mercrevive
skip_specs: true
owns:
  - docs/mercrevive.md
  - internal/merc/revive_log_test.go
  - openspec/changes/mercrevive/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "reserve 150->100, farm cap 16->18, need= in skip line plus gold wait line; offline tests only"
live: "not verified live"
```

- [x] work done
