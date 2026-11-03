# Tasks

```yaml
change: covarmor
status: archived
agent: covarmor
branch: agent/covarmor
worktree: d2lod-wt/covarmor
owns:
  - docs/research/covarmor.md
  - openspec/changes/covarmor/**
skip_specs: true
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0 hits"
result: "Data pass only (armor.txt and itemtypes.txt tables); no game.exe code run, no ledger rows, not verified live."
credit:
  author: owner
  agents: [Claude Code]
  consent: none
merge: 90b645a
```

## 1. Work

- [x] 1.1 Table census of armor.txt and itemtypes.txt (docs/research/covarmor.md)
- [ ] 1.2 Run game.exe defence and required-strength code under the emulator against d2d (not done)
