# Tasks

```yaml
change: chargedbolt
status: in-progress
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
result: "Verified on d2d (lab binary d2d.lab-debug-skills, slot 3, lockstep, throwaway Sorceress), compared to table, NOT game.exe. Charged Bolt PASS at levels 1, 4, 8, 13, 20: per-bolt mean 3.00, 4.52, 6.51, 9.00, 14.52 vs table 3.0, 4.5, 6.5, 9.0, 14.5, 42-120 hits per level, every single-bolt sample inside range +-0.5. Method: one frame per step, monsters life plus debug missiles bolt count; no new fork verb needed. writes after start: 0 by construction (d2d probe ledger has no runstart marker, U)."
```

## 1. Work

- [x] 1.1 Capacity check (verdict OK) and slot 3 free
- [x] 1.2 Per-frame method found (debug missiles + monsters per step)
- [x] 1.3 Five levels run, analysis script, evidence files
- [x] 1.4 docs/chargedbolt.md, card, OWNERS, merge-lines
