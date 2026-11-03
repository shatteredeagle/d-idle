# Tasks

```yaml
change: seeddist
status: archived
agent: seeddist
branch: agent/seeddist
worktree: ~/code/d2lod-wt/seeddist
skip_specs: true
owns:
  - openspec/changes/seeddist/**
  - docs/research/seeddist-trace.md
  - tools/emu/sd_*
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
  - "every V run number has stated cases, seeds and the test used"
result: "docs/tools only; see the doc for tags and limits"
merge: c34baaf
```

## 1. Work

- [x] 1.1 Read seed-study, d2d life / drop seeding, game.exe unit seeding (0x552df0, 0x45c3e0, 0x573f83)
- [x] 1.2 Game-side sampler (sd_game.py) and d2d-side sampler (sd_d2d_test.cpp, sd_build.sh)
- [x] 1.3 Tests (chi-square, KS, z) per quantity, 8 classes x >= 20,000 (sd_stats.py)
- [x] 1.4 Trace doc, ledger row F16 and regenerated rule-coverage.md, merge-lines.txt
