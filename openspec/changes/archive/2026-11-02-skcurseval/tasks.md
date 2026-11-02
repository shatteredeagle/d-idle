# Tasks

```yaml
change: skcurseval
status: archived
agent: skcurseval
branch: agent/skcurseval
worktree: ~/code/d2lod-wt/skcurseval
skip_specs: true
owns:
  - docs/research/skcurseval-trace.md
  - tools/emu/sk_curseval.py
  - tools/emu/sk_curseval_run.sh
  - tools/emu/sk_curseval_tbl.py
  - openspec/changes/skcurseval/**
exclusive: []
gates:
  - "every claim tagged V read / V run / D / S / I / U; numbers reproducible with command, case count and seed"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs only; first pass, not yet independently verified"
merge: 9584f9c
```

## 1. Work

- [x] 1.1 Oracle: game.exe FUN_00646ca0 on the 11 curse rows, levels 1..99, two seeds, vs d2d eval_calc
- [x] 1.2 Read the curse do for level and length divisor
- [x] 1.3 Trace doc (Q, D, B, limits, drill, matrix text), ledger line, coverage table regenerated
- [x] 1.4 Checks (changecheck, leakcheck)
