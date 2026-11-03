# Tasks

```yaml
change: abtool
status: archived
agent: abtool
branch: agent/abtool
worktree: d2lod-wt/abtool
skip_specs: false
owns:
  - tools/baseline/**
  - openspec/changes/abtool/**
exclusive: []
gates: python3 -I tools/baseline/tests/test_ab_compare.py passes; changecheck 0 warnings; leakcheck 0 hits
result: "docs/tools only; see the doc for tags and limits"
credit:
  author: owner
  agents: [Claude Code]
merge: e74797c
```

- [x] 1. ab_compare.py (ratio, interval, verdict, Poisson, regimes, pair correlation)
- [x] 2. Synthetic tests
- [x] 3. Delta spec for fidelity-gate
- [ ] 4. Adapter from the ops-repo record layout; reproduce pair_compare.py numbers on kitb/kitk2
