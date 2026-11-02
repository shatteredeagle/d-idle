# Tasks

```yaml
change: verifypaldur
status: archived
agent: verifypaldur
branch: agent/verifypaldur
worktree: d2lod-wt/verifypaldur
skip_specs: true
owns:
  - docs/research/verifypaldur.md
  - tools/emu/vp_wear.py
  - openspec/changes/verifypaldur/**
exclusive: []
gates: independent re-check of paldur-trace.md claims with an own model run against game.exe native code
result: "docs only; see the doc for tags, holds and limits"
merge: 7645230
```

- [x] 1. Read FUN_0057d3d0 / FUN_00559e30 in the disassembly
- [x] 2. Own model and own seeds against game.exe native run
- [x] 3. Write docs/research/verifypaldur.md
