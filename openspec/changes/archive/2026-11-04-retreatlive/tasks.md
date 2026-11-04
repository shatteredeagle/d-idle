# Tasks

```yaml
change: retreatlive
status: archived
agent: retreatlive
branch: agent/retreatlive
worktree: ~/code/d2lod-wt/retreatlive
skip_specs: true
owns:
  - docs/probes/retreat/**
  - docs/retreatlive.md
  - openspec/changes/retreatlive/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits"
result: "d2d live: RT.portal PASS, RT.drink chicken not drink (setup bound), RT.leave blocked"
live: "verified live on d2d (lab binary cheats3, slot 3)"
merge: cb2af00
```

- [x] Capacity check OK
- [x] RT.drink x2 on d2d
- [x] RT.portal x2 on d2d
- [x] Evidence under docs/probes/retreat/, findings in docs/retreatlive.md
- [x] changecheck, leakcheck
