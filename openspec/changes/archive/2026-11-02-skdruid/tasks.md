# Tasks

```yaml
change: skdruid
status: archived
agent: skdruid
branch: agent/skdruid
worktree: d2lod-wt/skdruid
skip_specs: true
owns:
  - openspec/changes/skdruid/**
  - docs/research/skdruid-trace.md
  - tools/emu/skdruid_calc.py
  - tools/emu/skdruid_test.cpp
  - tools/emu/skdruid_run.sh
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck -mode fail 0"
  - "ledger rows only with case counts"
result: "docs only; see the doc for tags and limits"
credit:
  author: owner
  agents: [Claude Code]
merge: 08590a5
```

## 1. Work

- [x] 1.1 Calc differential for the 30 Druid rows
- [x] 1.2 Trace doc, ledger row, coverage.py --write
- [ ] 1.3 Second pass: effects (forms, Twister / Tornado, Hurricane, Armageddon, summon spawn)
