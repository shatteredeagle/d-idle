# Tasks

```yaml
change: sorcdiff2
status: archived
agent: sorcdiff2
branch: agent/sorcdiff2
worktree: d2lod-wt/sorcdiff2
skip_specs: true
owns:
  - docs/research/sorcdiff2.md
  - tools/emu/sorcdiff2_*
  - openspec/changes/sorcdiff2/**
exclusive:
  - fork-branch:local/sorc-mana-timing-r2
  - fork-branch:local/sorc-armor-group-r2
  - fork-branch:local/sorc-bolts
gates:
  - "each local/sorc-* branch: harness and fix in separate commits on the newest upstream main, numeric noreply identity, private remote only"
  - "changecheck 0 new warnings, leakcheck 0 hits, merge_lines_check ok"
result: "docs and emulator tools only"
credit:
  component: mana payment timing of Sorceress casts (real-Fight harness, oracle, fix), Frozen Armor and Charged Bolt differentials
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: d5a82b5
```

## 1. Work

- [x] 1.1 game.exe's player mode scheduler and do: action frame tick, payment, interrupt (read and run)
- [x] 1.2 mana timing differential (Charged Bolt, Frost Nova, Fire Bolt, Frozen Armor): oracle, real-Fight harness, base vs fix counts
- [x] 1.3 fix branch local/sorc-mana-timing on the newest upstream main, pushed to the private remote
- [x] 1.4 armor group D2 reproduced and fixed; trigger D5 not built (no game-side harness)
- [x] 1.5 Charged Bolt pipeline: count, spawn, spread, hit, damage, seeds
- [x] 1.6 docs/research/sorcdiff2.md, merge-lines.txt, OWNERS lines
