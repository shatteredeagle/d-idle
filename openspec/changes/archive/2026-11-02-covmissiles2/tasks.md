# Tasks

```yaml
change: covmissiles2
status: archived
agent: covmissiles2
branch: agent/covmissiles2
worktree: ~/code/d2lod-wt/covmissiles2
skip_specs: true
owns:
  - docs/research/covmissiles2-trace.md
  - tools/emu/cm2_run.sh
  - tools/emu/cm2_game.py
  - tools/emu/cm2_probe.cpp
  - tools/emu/cm2_cmp.py
  - docs/coverage/ledger.tsv
  - docs/coverage/rule-coverage.md
  - openspec/changes/covmissiles2/**
exclusive: []
gates:
  - "oracle: game.exe record vs the real d2d build, counted cases, seeds stated; no d2d change"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; coverage.py --check"
result: "docs/tools only; see the doc for tags and limits"
merge: 41db496
```

## 1. Work

- [x] 1.1 Pick the rows (Act 1 / 2 skill-sourced monster missiles not claimed in ledger.tsv)
- [x] 1.2 Game side, d2d probe and compare; two seeds
- [x] 1.3 Trace doc (Q, D, B, drill recipe, matrix text)
- [x] 1.4 Ledger rows 22, 24, 32, 77, 130, 346, 653; regenerate rule-coverage.md

## 2. Open

- [ ] 2.1 Build local/missilepoison-skillrows and run cm2_run.sh against it for the "after" count
- [ ] 2.2 Carried-skill rows (62, 96, 119, 133) on the skill path; the skill level a monster missile gets
