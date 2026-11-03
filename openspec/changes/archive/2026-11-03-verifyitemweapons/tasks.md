# Tasks

```yaml
change: verifyitemweapons
status: archived
agent: verifyitemweapons
branch: agent/verifyitemweapons
worktree: d2lod-wt/verifyitemweapons
skip_specs: true
owns:
  - docs/research/verifyitemweapons.md
  - tools/emu/viw_*
  - openspec/changes/verifyitemweapons/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs and a table recount only; two wrong statements found in the first pass, no game.exe claim to verify; not verified live"
credit:
  author: owner
  agents: [Claude Code]
merge: f4ae491
```

- [x] recount weapons.txt facts with an own script
- [x] verdict per claim, ledger check
- [x] trace doc, merge-lines
