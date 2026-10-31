# Tasks

```yaml
change: spec-backfill-combat
status: archived
agent: specfix
branch: agent/specfix
worktree: ~/code/d2lod-wt/specfix
owns:
  - openspec/changes/spec-backfill-combat/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
merge: 30ddfab
result: "Delta spec for capability combat-loop written from internal/combat code and tests (navstall); docs and specs only, nothing run live."
```

## 1. Work

- [x] 1.1 Delta ADDED Requirements for combat-loop: failed walk tick, no retreat while pinned, no-progress watchdog, champion caution (card champ, merged), each checked against code and test
- [x] 1.2 validate --all --strict and changecheck pass
