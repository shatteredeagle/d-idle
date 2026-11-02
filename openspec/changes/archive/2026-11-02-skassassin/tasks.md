# Tasks

```yaml
change: skassassin
status: archived
agent: skassassin
branch: agent/skassassin
worktree: d2lod-wt/skassassin
skip_specs: true
owns:
  - docs/research/skassassin-trace.md
  - openspec/changes/skassassin/**
exclusive: []
gates: leakcheck 0 hits, changecheck 0 warnings
result: "docs only; see the doc for tags and limits"
credit:
  component: SKASSASSIN
  author: owner
  agents: [Claude Code]
merge: de908e2
```

- [x] 1. Dump skills table, group ass rows
- [x] 2. Write trace
- [ ] 3. Run SKMANA harness on ids 251-280 (next pass)
