# Tasks

```yaml
change: spec-backfill-d2dexit
status: in-progress
agent: specback
branch: agent/specback
worktree: ~/code/d2lod-wt/specback
owns:
  - openspec/changes/spec-backfill-d2dexit/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
merge:
result:
```

## 1. Work

- [x] 1.1 Delta for backend-contract written from d2dworld, d2denv and d2dsrc code and tests; every requirement checked against the current code and the named tests
- [x] 1.2 validate --all --strict and changecheck pass
