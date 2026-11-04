# Tasks

```yaml
change: retreatlive
status: in-progress
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
result: "RT.portal PASS (2 identical runs, d2d). RT.drink FAIL as designed: 20 percent life is below chickenPct 25, the bot correctly chooses chicken; bound question, propose 40 percent. Verified live on d2d, not game.exe."
live: "verified live on d2d (lab binary cheats3, slot 3)"
```

- [x] Capacity check OK
- [x] RT.drink x2 on d2d
- [x] RT.portal x2 on d2d
- [x] Evidence under docs/probes/retreat/, findings in docs/retreatlive.md
- [x] changecheck, leakcheck
