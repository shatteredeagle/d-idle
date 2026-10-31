# Tasks

```yaml
change: d2dsoak
status: archived
agent: d2dsoak
branch: agent/d2dsoak
worktree: ~/code/d2lod-wt/d2dsoak
owns:
  - cmd/d2dsoak/**
  - docs/d2d-soak.md
  - internal/d2dsoakstat/**
  - tools/d2dsoak/**
exclusive: []
gates:
  - "merged with tests passing, see 1a626bc"
merge: 1a626bc
result: "dry run live V 2026-10-30 09:11 (lock, start, 3 virtual minutes, quit, no d2d left); real bot run U (waited for the d2d backend)"
```

## 1. Work

- [x] 1.1 Implemented and merged
