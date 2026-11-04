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
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: ""
live: "not verified live"
```

- [ ] work done
