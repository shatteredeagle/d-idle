# Tasks

```yaml
change: skground
status: archived
agent: skground
branch: agent/skground
worktree: ~/code/d2lod-wt/skground
skip_specs: true
owns:
  - docs/research/sk-ground-trace.md
  - docs/research/skground-trace.md
  - tools/emu/sk_ground*
  - tools/emu/skg_*
  - openspec/changes/skground/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs only; first pass, not yet independently verified"
live: "docs only, nothing run live"
merge: 650c59b
credit:
  component: Standing missile footprint and burst roll (ground), trace
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifybundle, own harness)"]
  consent: none
```

- [x] Meteor blast read (FUN_005aabb0, FUN_0056bad0, FUN_0056b7e0, FUN_0056b3e0, FUN_006492a0)
- [x] game-side native run (shape, rooms, filter, radius) and real-d2d differential (120,000 cases)
- [x] deviations M1-M4, bot findings, drill recipe, matrix cells
- [x] coverage ledger rows for Meteor (56) and meteorcenter (101); merge-lines.txt
