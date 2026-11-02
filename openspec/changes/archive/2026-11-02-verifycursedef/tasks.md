# Tasks

```yaml
change: verifycursedef
status: archived
agent: verifycursedef
branch: agent/verifycursedef
worktree: d2lod-wt/verifycursedef
skip_specs: true
owns:
  - docs/research/verify-cursedef.md
  - tools/emu/vcd_calc.py
  - tools/emu/vcd_test.cpp
  - tools/emu/vcd_run.sh
  - tools/emu/vcd_tbl.py
  - openspec/changes/verifycursedef/**
exclusive: []
gates:
  - "own game.exe oracle and own d2d probe; counts and seeds given"
  - "changecheck 0 warnings; leakcheck -mode fail 0"
result: "docs and emulator tools only"
merge: 86a7133
```

## 1. Work

- [x] 1.1 SKCURSEVAL: own FUN_00646ca0 oracle + own eval_calc probe, two seeds, break inputs
- [x] 1.2 SKDEFENDERS: own read of FUN_0057be00 and d2d's panel / monster_blow
- [x] 1.3 SKSHIELD: own read of the equip attach FUN_004c0d20
- [x] 1.4 Verdict doc, checks
