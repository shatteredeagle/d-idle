# Tasks

```yaml
change: spec-backfill-botloop
status: archived
agent: specfix
branch: agent/specfix
worktree: ~/code/d2lod-wt/specfix
owns:
  - openspec/changes/spec-backfill-botloop/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
merge: c14f9c7
result: ADDED deltas applied to openspec/specs by archive; offline, spec text only, not verified live
```

## 1. Work

- [x] 1.1 Delta ADDED Requirements for bot-loop (restarts count, per-game forgetting, monster classification) checked against internal/bot and internal/d2 code and tests
- [x] 1.2 validate --all --strict and changecheck pass
