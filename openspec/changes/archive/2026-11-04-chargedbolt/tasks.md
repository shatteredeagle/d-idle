# Tasks

```yaml
change: chargedbolt
status: archived
agent: chargedbolt
branch: agent/chargedbolt
worktree: ~/code/d2lod-wt/chargedbolt
skip_specs: true
owns:
  - openspec/changes/chargedbolt/**
  - docs/chargedbolt.md
  - docs/probes/sksmoke/2026-11-04-SK-cb*
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "PASS on d2d, per-bolt means within 0.5 percent; not game.exe"
merge: 10491e7
```

## 1. Work

- [x] 1.1 Capacity check (verdict OK) and slot 3 free
- [x] 1.2 Per-frame method found (debug missiles + monsters per step)
- [x] 1.3 Five levels run, analysis script, evidence files
- [x] 1.4 docs/chargedbolt.md, card, OWNERS, merge-lines
