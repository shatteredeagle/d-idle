# Tasks

```yaml
change: verifylevelsareas
status: in-progress
agent: verifylevelsareas
branch: agent/verifylevelsareas
worktree: d2lod-wt/verifylevelsareas
skip_specs: true
owns:
  - docs/research/verifylevelsareas.md
  - openspec/changes/verifylevelsareas/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs only; lv_area.py re-run (1,128 cases, 6 id-0 diffs); oracle is the table, d2d side read only; no live game"
credit:
  author: owner
  agents: [Claude Code]
```

- [x] re-run lv_area.py on the worker host
- [x] read d2d area_level users
- [x] verdicts, gaps, ledger judgement
- [x] merge-lines
