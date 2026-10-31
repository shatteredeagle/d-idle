# Tasks

```yaml
change: origprobes
status: archived
agent: origprobes
branch: agent/origprobes
worktree: ~/code/d2lod-wt/origprobes
owns:
  - docs/probes/orig-spawns.csv
  - docs/probes/orig-casts.csv
  - docs/orig-spawns.md
  - docs/orig-casts.md
  - cmd/origcasts/**
exclusive:
  - instance:i0
gates:
  - "every number is tagged V-orig (measured on Game.exe) or U"
merge: 753ff01
result: 'original-side probes: monsters per visit orig vs d2d: Blood Moor 90/69, Cold Plains 173/127, Den 56/54, Stony Field 131/91, Dark Wood 175/126 (original counts are a lower bound: activated rooms only); champions per visit Stony Field 2.75 vs 0.95; Corpsefire not seen 4/4 in the original (U); casts per kill Fallen skill 1/4/8 original 0.96/0.90/0.69 vs d2d commands 2.59/1.18/0.97 (d2d count was commands, not casts: see gapcasts); frame rate ~23.5 fps estimate U; Testy now clvl 9, Charged Bolt 8 via ledger cheats'
```

## 1. Work

- [x] 1.1 spawns: Blood Moor, Den of Evil, Cold Plains, Stony Field, Dark Wood on fresh games (orig-spawns.csv, orig-spawns.md)
- [x] 1.2 casts per kill, Charged Bolt vs Fallen at skill 1/4/8 (orig-casts.csv, orig-casts.md)
- [x] 1.3 frame-rate estimate (mana regeneration)
