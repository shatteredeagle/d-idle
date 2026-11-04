# Tasks

```yaml
change: merchire
status: in-progress
agent: merchire
branch: agent/merchire
worktree: ~/code/d2lod-wt/merchire
skip_specs: true
owns:
  - docs/merchire.md
  - internal/drill/merchire_test.go
  - openspec/changes/merchire/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: ""
live: "not verified live"
```

- [ ] work done
