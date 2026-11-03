# Tasks

```yaml
change: stashfield
status: archived
agent: stashfield
branch: agent/stashfield
worktree: ~/code/d2lod-wt/stashfield
skip_specs: true
owns:
  - internal/d2dsoakstat/invariants.go
  - internal/d2dsoakstat/invariants_test.go
  - internal/d2dsoakstat/invreport.go
  - internal/d2dsoakstat/minute.go
  - internal/bot/observe.go
  - docs/research/stashfield.md
  - docs/geninv.md
  - docs/d2d-soak.md
  - openspec/changes/stashfield/**
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dsoakstat ./internal/bot ./cmd/d2dsoak; changecheck -cli 0 warnings; leakcheck 0 hits"
merge: 875cd7c
result: "whole repo vet and test pass on the worker host; changecheck overlaps with the stale specmin card"
credit:
  author: owner
  agents: [Claude Code]
```

## 1. Work

- [x] 1.1 minute line gets additive `stashGold` (internal/bot/observe.go); verified by compile and vet only
- [x] 1.2 d2dsoakstat parses it (Minute.StashGold, HasStash); BELT semantics unchanged
- [x] 1.3 report-only belt window gold line with a named PotionPrice constant; tests for poverty, funded, old archive
- [x] 1.4 docs: geninv.md, d2d-soak.md, docs/research/stashfield.md
