# Tasks

```yaml
change: skdefenders
status: archived
agent: skdefenders
branch: agent/skdefenders
worktree: ~/code/d2lod-wt/skdefenders
skip_specs: true
owns:
  - openspec/changes/skdefenders/**
  - docs/research/skdefenders-trace.md
  - tools/emu/skdef_*
  - docs/coverage/ledger.tsv
  - docs/coverage/rule-coverage.md
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
  - "every number tagged V run is reproducible with the commands in the trace (cases and seeds given)"
result: "docs only; first pass, not yet independently verified"
merge: 51a5d6e
```

## 1. Work

- [x] 1.1 Read FUN_0057be00, FUN_0057bf80, FUN_0057bf10, FUN_0057c1e0, FUN_0057c060, FUN_0063ee90, FUN_00572840, FUN_00573cb0 (V read)
- [x] 1.2 Oracle skdef_resist.py (game.exe natively) and skdef_test.cpp (real rules::monster_blow); rule and absorb checks
- [x] 1.3 Differential on two seeds, 100,000 random cases plus the grid each
- [x] 1.4 Trace doc, ledger row, rule-coverage regenerated, merge lines
- [ ] 1.5 Live drills (recipe in the trace, section 8): not run here
