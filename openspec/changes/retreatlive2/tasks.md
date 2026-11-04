# Tasks

```yaml
change: retreatlive2
status: in-progress
agent: retreatlive2
branch: agent/retreatlive2
worktree: ~/code/d2lod-wt/retreatlive2
skip_specs: true
owns:
  - docs/probes/retreat/2026-11-04-RT2*
  - docs/retreatlive2.md
  - openspec/changes/retreatlive2/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits"
result: "verified live on d2d (lab binary cheats3, slot 3): RT.drink PASS (drink-life, belt 4 to 3, 4 ticks to effect), RT.portal PASS (1 tick, bound 10); both identical in 2 runs"
```

- [x] Capacity check OK
- [x] RT.drink x2 on d2d
- [x] RT.portal x2 on d2d
- [x] Evidence and docs/retreatlive2.md
- [x] changecheck, leakcheck
