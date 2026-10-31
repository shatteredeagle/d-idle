# Tasks

```yaml
change: soakfix
status: archived
agent: soakfix
branch: agent/soakfix
worktree: ~/code/d2lod-wt/soakfix
owns:
  - cmd/d2dsoak/**
  - internal/d2dsoakstat/**
  - openspec/changes/soakfix/**
exclusive: []
gates:
  - "go test -p 1 ./cmd/d2dsoak ./internal/d2dsoakstat"
merge: 4d0c854
result: "offline only, not verified live; table tests per finding"
```

## 1. Work
- [x] exit codes incl. -json, -exit-zero, docs/d2d-soak.md 3c
- [x] invalid/short/early-ended seeds excluded
- [x] min sample INCONCLUSIVE, NaN guards, duplicate seeds, parse counting, jump cap
- [x] pacer note written up (not changed)
