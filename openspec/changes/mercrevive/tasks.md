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
result: ""
live: "not verified live"
```

- [ ] work done
