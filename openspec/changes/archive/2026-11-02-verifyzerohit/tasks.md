# Tasks

```yaml
change: verifyzerohit
status: archived
agent: verifyzerohit
branch: agent/verifyzerohit
worktree: d2lod-wt/verifyzerohit
skip_specs: true
owns:
  - docs/research/verifyzerohit.md
  - tools/emu/vzh_*
  - openspec/changes/verifyzerohit/**
exclusive: []
gates:
  - own disassembly reading and model before the authors' scripts are read
  - own game.exe emulator runs with own seeds, 0 differences against the own model
result: "docs only; see the doc for tags and limits"
credit:
  component: SKZEROHIT
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
merge: 873fd1a
```

- [x] 1 Own disassembly reading of FUN_0057cee0, FUN_0057cb00, FUN_005a4270, FUN_005a2ba0, FUN_005a29a0, FUN_005ad730, FUN_0057c6c0
- [x] 2 Own harness tools/emu/vzh_run.py, seeds 101, 202, 303: 0 differences
- [x] 3 Event-2 pointer search (tools/emu/vzh_xref.py)
- [x] 4 Claim-by-claim verdict for both passes, ledger-row assessment
- [ ] 5 d2d side run (not done: read only)
- [x] 6 changecheck and leakcheck
