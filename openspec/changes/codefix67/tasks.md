# Tasks

```yaml
change: codefix67
status: in-progress
agent: codefix67
branch: agent/codefix67
worktree: ~/code/d2lod-wt/codefix67
skip_specs: true
owns:
  - docs/codefix67.md
  - openspec/changes/codefix67/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: ""
live: "not verified live"
```

- [ ] work done
