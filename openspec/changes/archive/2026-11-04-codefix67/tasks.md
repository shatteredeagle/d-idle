# Tasks

```yaml
change: codefix67
status: archived
agent: codefix67
branch: agent/codefix67
worktree: ~/code/d2lod-wt/codefix67
skip_specs: true
owns:
  - docs/codefix67.md
  - openspec/changes/codefix67/**
  - internal/town/codefix67_test.go
  - internal/bot/codefix67_test.go
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "offline tests fail without the fix; not verified live"
live: "not verified live"
merge: 6f679c9
```

- [x] work done
