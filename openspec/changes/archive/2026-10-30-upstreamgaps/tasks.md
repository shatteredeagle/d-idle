# Tasks

```yaml
change: upstreamgaps
status: archived
agent: upstreamgaps
branch: agent/upstreamgaps
worktree: ~/code/d2lod-wt/upstreamgaps
owns:
  - docs/research/upstream-gaps/**
  - docs/research/upstream-pr-candidates.md
exclusive:
  - upstream-pr:missile-damage-fine
gates:
  - "each item has evidence, a verdict and a draft when it differs"
  - "nothing pushed or posted"
merge: 1a67348
result: 'Charged Bolt: bolts/ranges equal game.exe (emulator, 3000 cases); d2d truncates spell damage to whole points (roll >> 8; game.exe keeps 256ths: -17 % at skill 1): PR candidate (prbuild); Act 1 monster HP matches (2202/2202); d2d monster placement equals game.exe per room: density gap was our sweep coverage 58-71 %'
```

## 1. Work

- [x] 1.1 item 1: Charged Bolt hit and damage trace, oracle script, verdict (differs: whole-point damage)
- [x] 1.2 item 2: Act 1 monster HP, formula checked by upstream's oracle, re-run
- [x] 1.3 item 3: spawn counts, DRLG generator equals game.exe, deficit is downstream
- [x] 1.4 candidates draft document
