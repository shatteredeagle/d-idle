# Tasks

```yaml
change: skiasgame2
status: archived
agent: skiasgame2
branch: agent/skiasgame2
worktree: ~/code/d2lod-wt/skiasgame2
skip_specs: true
owns:
  - docs/research/skiasgame2-trace.md
  - tools/emu/skiasgame2_rate.py
  - docs/coverage/ledger.tsv
  - docs/coverage/rule-coverage.md
  - openspec/changes/skiasgame2/**
exclusive: []
gates: differential game.exe attack rate vs d2d attack_speed, 0 mismatches over the IAS x WSM grid (counts in the trace)
result: "docs only; first or second pass, not yet independently verified"
merge: 0430f68
```

- [x] 1. Read the attack branch of FUN_00623f50 and its helpers
- [x] 2. Run it under the emulator against d2d's arithmetic (grid, sias, ticks, modes, shapeshift, skill modes)
- [x] 3. Write docs/research/skiasgame2-trace.md
- [x] 4. Ledger row F07 and regenerate rule-coverage.md
