# Tasks

```yaml
change: hirereader
status: archived
agent: hirereader
branch: agent/hirereader
worktree: ~/code/d2lod-wt/hirereader
skip_specs: true
owns:
  - docs/research/hirereader-trace.md
  - docs/research/hirereader-design.md
  - tools/emu/hrd_pay.py
  - tools/emu/hrd_d2d_test.cpp
  - tools/emu/hrd_build.sh
  - tools/emu/hrd_run.sh
  - openspec/changes/hirereader/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
  - "a V run claim needs counts: purse differential 200,000 cases, 0 differ"
result: "docs/tools only"
live: "docs only, nothing run live"
merge: 3f3db38
```

- [x] read the fork and bot sides (dump merc, state, menu, hire and resurrect commands)
- [x] purse differential (hrd_*): 200,000 cases, 0 differ
- [x] trace and design docs
- [x] card, OWNERS, merge lines, checks
