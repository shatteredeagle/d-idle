# Tasks

```yaml
change: palaura
status: archived
agent: palaura
branch: agent/palaura
worktree: ~/code/d2lod-wt/palaura
skip_specs: true
owns:
  - docs/research/palaura-trace.md
  - tools/emu/palaura_*
  - openspec/changes/palaura/**
exclusive: []
gates:
  - "game.exe's own aura code run under the emulator on real player and monster units (area search, filter, apply, enemy hit, Prayer mana and state 85, regeneration), every claim tagged"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict"
result: "docs only; first or second pass, not yet independently verified"
merge: fcd78e5
credit:
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifypalaura, independent run)"]
  consent: none
```

## 1. Work

- [x] 1.1 Read FUN_0056b7e0, FUN_0056b3e0, FUN_005cedc0, FUN_005cf010, FUN_005cf3a0, FUN_005806f0 and helpers
- [x] 1.2 Emulator runs: area radius, filter, apply, enemy hit, Prayer cases, regeneration, town, line of sight
- [x] 1.3 d2d's version (source), deviations, bot findings
- [x] 1.4 Trace doc, drill recipe, matrix text, merge lines
