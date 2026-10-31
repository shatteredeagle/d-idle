# Tasks

```yaml
change: spec-backfill-viewer
status: in-progress
agent: specfix
branch: agent/specfix
worktree: ~/code/d2lod-wt/specfix
owns:
  - openspec/changes/spec-backfill-viewer/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
merge:
result: "Delta spec for capability dashboard-pub written from cmd/d2dview code and tests; timings in browsers are UNVERIFIED, only the 10 minute non-black live run is verified."
```

## 1. Work

- [x] 1.1 Delta ADDED Requirements for dashboard-pub (backend, engine, status row, image reconnect) checked against cmd/d2dview
- [x] 1.2 validate --all --strict and changecheck pass
