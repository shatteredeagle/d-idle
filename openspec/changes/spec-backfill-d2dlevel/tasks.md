# Tasks

```yaml
change: spec-backfill-d2dlevel
status: in-progress
agent: specback
branch: agent/specback
worktree: ~/code/d2lod-wt/specback
owns:
  - openspec/changes/spec-backfill-d2dlevel/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
merge:
result:
```

## 1. Work

- [x] 1.1 Delta for early-game-policy written from internal/bot goal.go and denwaive_test.go; every requirement checked against the current code and the named tests
- [x] 1.2 validate --all --strict and changecheck pass
