# Tasks

```yaml
change: itemweapons
status: archived
agent: itemweapons
branch: agent/itemweapons
worktree: d2lod-wt/itemweapons
skip_specs: true
owns:
  - docs/research/itemweapons-trace.md
  - openspec/changes/itemweapons/**
exclusive: []
gates: trace doc exists, leakcheck and changecheck clean
result: "docs only; see the doc for tags and limits"
credit:
  author: owner
  agents: [Claude Code]
merge: 5c44a78
```

- [x] 1. Dump weapons.txt and tabulate (D)
- [ ] 2. Run game.exe damage and speed formulas against d2d (not done)
- [ ] 3. itemtypes and armor tables
