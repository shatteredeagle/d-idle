# Tasks

```yaml
change: riskmap
status: archived
agent: riskmap
branch: agent/riskmap
worktree: ~/code/d2lod-wt/riskmap
owns:
  - docs/research/d2d-fidelity-riskmap.md
  - tools/riskmap/**
exclusive: []
gates:
  - "python3 -I -m unittest in tools/riskmap passes"
  - "go run ./cmd/changecheck: 0 warnings"
merge:
result: "docs and script only; upstream data fetched 2026-10-30 (commit 2a04c18e); nothing run live, nothing posted to GitHub"
merge: a4b9cd3
result: "ranked fidelity risk map, top 10 tasks ~52 agent-h, SME reviewed"
```

## 1. Work

- [x] 1.1 fetch the upstream gap data (gaps.json, functions.tsv, stubs.md)
- [x] 1.2 map our metric dependencies to upstream modules, functions and listed gaps
- [x] 1.3 rank by impact x (1 - confidence), top 10 tasks with a first step
- [x] 1.4 write docs/research/d2d-fidelity-riskmap.md and tools/riskmap (script, dependencies.json, fixture test)
