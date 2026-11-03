# Tasks

```yaml
change: verifystateev2
status: archived
agent: verifystateev2
branch: agent/verifystateev2
worktree: d2lod-wt/verifystateev2
skip_specs: true
owns:
  - docs/research/verifystateev2.md
  - tools/emu/vse_*
  - openspec/changes/verifystateev2/**
exclusive: []
gates:
  - own disassembly reading and model before the authors' scripts are read
  - own game.exe emulator runs with own seeds (4242, 90210), game-vs-game oracle, 0 exceptions on the verified claims
result: "docs only; see the doc for tags and limits"
credit:
  component: SKSTATEEV
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
merge: 8dd3f97
```

- [x] 1 Own disassembly reading of FUN_0057dbf0, FUN_0057d4f0, FUN_005c0c30, FUN_0057c1e0, FUN_0057c6c0, FUN_0057b7d0 (tools/emu/vse_dis.py)
- [x] 2 Own harness tools/emu/vse_run.py (X0..X3), seeds 4242 and 90210
- [x] 3 Claim-by-claim verdicts for both passes, deviations, ledger-row assessment
- [ ] 4 d2d side run (not done: read only) and ES node FUN_005ca6c0 run (not done)
- [x] 5 changecheck and leakcheck
