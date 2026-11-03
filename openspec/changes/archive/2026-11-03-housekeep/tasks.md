# Tasks

```yaml
change: housekeep
status: archived
agent: housekeep
branch: agent/housekeep
worktree: d2lod-wt/housekeep
skip_specs: true
owns:
  - openspec/changes/housekeep/**
  - docs/research/housekeep-20261103.md
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck 0 hits; tools/ci/check.sh ok"
merge: 704deff
result: "docs only"
credit:
  author: owner
  agents: [Claude Code]
```

## 1. Work

- [x] 1.1 archive soak-minutes-virtual (merged 12c11b9), apply its soak-gates delta, drop its OWNERS rows
- [x] 1.2 read-only inventory docs/research/housekeep-20261103.md
- [ ] 1.3 msvcc2026-trace.md correction banner: the file only exists on the unmerged agent/msvcc2026 branch, text is in the inventory doc (owner of that branch applies it)
