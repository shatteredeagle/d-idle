# Tasks

```yaml
change: skstateev
status: archived
agent: skstateev
branch: agent/skstateev
worktree: ~/code/d2lod-wt/skstateev
skip_specs: true
owns:
  - docs/research/sk-state-events-trace.md
  - tools/emu/sk_stateev_xref.py
  - openspec/changes/skstateev/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs only; see the doc for tags, holds and limits"
live: "docs only, nothing run live"
merge: 34d019b
```

- [x] find the dispatcher and its call sites
- [x] answer hit/miss and absorb order (V read)
- [ ] native run of FUN_0057d4f0 / FUN_0057c1e0 (open, U)
