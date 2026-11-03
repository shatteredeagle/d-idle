# Tasks

```yaml
change: xphshort
status: archived
agent: xphshort
branch: agent/xphshort
worktree: d2lod-wt/xphshort
skip_specs: true
owns:
  - docs/research/xphshort.md
  - tools/baseline/xph_budget.py
  - tools/baseline/tests/test_xph_budget.py
  - openspec/changes/xphshort/**
exclusive: []
gates: python3 tools/baseline/tests/test_xph_budget.py; changecheck 0 new warnings; leakcheck 0 hits
result: "python tests pass; docs and tooling only"
credit:
  author: owner
  agents: [Claude Code]
  consent: none
merge: e1c4f73
```

- [x] 1. xph_budget.py (budget per band, decomposition, horizon) and fixture test
- [x] 2. Budget, split, verdict on the fixes, remedies in docs/research/xphshort.md
- [x] 3. Multi-seed Gate 1 plan and Gate 2 schedule
