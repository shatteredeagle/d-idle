# Tasks

```yaml
change: spec-backfill-rejuvuse
status: archived
agent: specback
branch: agent/specback
worktree: ~/code/d2lod-wt/specback
owns:
  - openspec/changes/spec-backfill-rejuvuse/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
merge: c14f9c7
result: ADDED deltas applied to openspec/specs by archive; offline, spec text only, not verified live
```

## 1. Work

- [x] 1.1 Deltas for combat-loop and early-game-policy written from resource, loot and town code and tests; every requirement checked against the current code and the named tests
- [x] 1.2 validate --all --strict and changecheck pass
