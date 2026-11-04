# Tasks

```yaml
change: noareacards
status: archived
agent: noareacards
branch: agent/noareacards
worktree: ~/code/d2lod-wt/noareacards
skip_specs: true
owns:
  - docs/noareacards.md
  - openspec/changes/noareacards/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits; merge_lines_check ok"
result: "docs only; no code; awaits the user review of each card"
live: "not verified live (docs only)"
merge: f968460
```

- [x] work done
