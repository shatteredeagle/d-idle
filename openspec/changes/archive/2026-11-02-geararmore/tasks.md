# Tasks

```yaml
change: geararmore
status: archived
agent: pa-geararmore
branch: agent/geararmore
worktree: ~/code/d2lod-wt/geararmore
skip_specs: true
owns:
  - openspec/changes/geararmore/**
  - docs/research/geararmore.md
  - tools/emu/gar_run.py
  - tools/emu/gar_dis.py
  - tools/emu/gar_census.sh
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
  - "every V run number has a command, a case count and a seed in the doc"
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
merge: eba3926
```

## 1. Work

- [x] 1.1 Read FUN_0057d9b0 (stat 179 loop) and FUN_0057b7d0 (stat 120) in the disassembly
- [x] 1.2 gar_run.py parts a and b, 100,000 random + boundaries each, seed 7
- [x] 1.3 Table census gar_census.sh
- [x] 1.4 Doc docs/research/geararmore.md, card, OWNERS, merge-lines
