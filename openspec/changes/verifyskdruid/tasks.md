# Tasks

```yaml
change: verifyskdruid
status: in-progress
agent: verifyskdruid
branch: agent/verifyskdruid
worktree: d2lod-wt/verifyskdruid
skip_specs: true
owns:
  - docs/research/verifyskdruid.md
  - tools/emu/vsd_*
  - openspec/changes/verifyskdruid/**
exclusive: []
gates:
  - own disassembly reading and model before the authors' scripts are read
  - own game.exe emulator runs with own seeds, 0 differences against d2d eval_calc, plus a negative control that differs
result: "docs and emulator scripts only, not verified live; calc aspect V run (30,680 own cases DIFF 0, blvl/lvl split, negative control DIFF 78); first pass holds with limits, five value descriptions and the bot finding wrong; no deviation, no PR candidate"
credit:
  component: SKDRUID
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
```

- [x] 1 Own disassembly reading of FUN_00646ca0, 0x646be0, FUN_00646460, FUN_00646c00, FUN_006442a0
- [x] 2 Own harness tools/emu/vsd_* with blvl / lvl split, seeds 101 and 202: DIFF 0; negative control DIFF 78
- [x] 3 Authors' seed 1 re-run: DIFF 0
- [x] 4 Claim-by-claim verdicts, bot findings, ledger-row assessment
- [ ] 5 Effects (forms, Twister / Tornado, storms, spawn path): not run by anyone
- [x] 6 changecheck and leakcheck
