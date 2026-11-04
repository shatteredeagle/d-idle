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
  - internal/town/codefix67_test.go
  - internal/bot/codefix67_test.go
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "F6 and F7 fixed offline with fake tests (fail without fix by hand revert); touched small edits in internal/town/basicsfix.go and internal/bot/goal.go, observe.go, resource.go; not verified live"
live: "not verified live"
```

- [x] work done
