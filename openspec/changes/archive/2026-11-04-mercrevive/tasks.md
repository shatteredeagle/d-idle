# Tasks

```yaml
change: mercrevive
status: archived
agent: mercrevive
branch: agent/mercrevive
worktree: ~/code/d2lod-wt/mercrevive
skip_specs: true
owns:
  - docs/mercrevive.md
  - openspec/changes/mercrevive/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "offline tests, two kits; not verified live"
live: "not verified live"
merge: ea07d65
```

- [x] work done
