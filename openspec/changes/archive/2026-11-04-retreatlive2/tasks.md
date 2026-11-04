# Tasks

```yaml
change: retreatlive2
status: archived
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
result: "d2d live: both PASS, runs identical; RT.leave blocked; not game.exe"
merge: 5569f16
```

- [x] Capacity check OK
- [x] RT.drink x2 on d2d
- [x] RT.portal x2 on d2d
- [x] Evidence and docs/retreatlive2.md
- [x] changecheck, leakcheck
