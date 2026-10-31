# Tasks

```yaml
change: spec-backfill-d2dcombat
status: in-progress
agent: specback
branch: agent/specback
worktree: ~/code/d2lod-wt/specback
owns:
  - openspec/changes/spec-backfill-d2dcombat/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
merge:
result:
```

## 1. Work

- [x] 1.1 Delta for backend-contract written from internal/d2dcombat code and tests; every requirement checked against the current code and the named tests
- [x] 1.2 validate --all --strict and changecheck pass
