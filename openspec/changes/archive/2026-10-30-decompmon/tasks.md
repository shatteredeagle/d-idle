# Tasks

```yaml
change: decompmon
status: archived
merge: 91d28c1
agent: decompmon
branch: agent/decompmon
worktree: ~/code/d2lod-wt/decompmon
owns:
  - docs/research/monster-attack-trace.md
  - docs/research/upstream-pr-candidates-monster.md
  - tools/emu/monster_*.py
  - openspec/changes/decompmon/**
exclusive: []
gates:
  - "every rule in the trace doc is tagged V (game.exe read or run under unicorn), D, S or U with an address"
result: "melee hit chance, damage roll and application traced and oracle-checked (100k cases each, 0 differ from d2d's rules); one deviation: the get-hit rule (game.exe FUN_0057cb00 tiered by hit class, d2d a fixed twelfth / eighth), reported as a finding, PROOF incomplete, no PR branch. Not verified live."
```

## 1. Work

- [x] 1.1 Read DECOMPXP / DECOMPSPAWN / UPSTREAMGAPS docs and upstream combat.md, monsters.md, monster-ai.md
- [x] 1.2 Melee path in game.exe: FUN_0057ec10, FUN_0057d9b0, FUN_0057b420, FUN_0057c6c0, FUN_0057cee0, FUN_0057cb00 (disassembly, V)
- [x] 1.3 Ranged / spell / enchants: LE bolts, FE / CE, Shaman fire, spikes, arrows (S + D tables)
- [x] 1.4 d2d comparison item by item; oracle `tools/emu/monster_hit.py` (3 x 100k random + 672 boundary cases)
- [x] 1.5 Death-risk model vs docs/plan/death-data.md and docs/g1-log.md
- [x] 1.6 Trace doc, PR candidate doc (finding only), changecheck 0 warnings
