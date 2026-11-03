# Tasks

```yaml
change: verifyskassassin
status: in-progress
agent: verifyskassassin
branch: agent/verifyskassassin
worktree: d2lod-wt/verifyskassassin
skip_specs: true
owns:
  - docs/research/verifyskassassin.md
  - tools/emu/vsa_*
  - openspec/changes/verifyskassassin/**
exclusive: []
gates:
  - own disassembly reading and model before the authors' material is compared
  - own game.exe emulator runs with own seed, 0 differences against the own model
result: "docs only, not verified live; mana check and payment for ids 251..280 V run (game.exe 0 model mismatches, game.exe vs d2d 0 differences in 39,305 cases); first pass wrong on srvstartfunc and on the rule attribution; second pass not found"
credit:
  component: SKASSASSIN
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
```

- [x] 1 Own disassembly reading of FUN_0056c160 and FUN_0056bfe0 (tools/emu/vsa_dis.py)
- [x] 2 Own oracle tools/emu/vsa_mana.py, seed 20261102: 0 model mismatches
- [x] 3 game.exe vs d2d differential tools/emu/vsa_diff.sh: 0 differences
- [x] 4 Claim-by-claim verdict, ledger assessment
- [ ] 5 Second pass (not found in this clone)
- [ ] 6 Charge-up, trap and shadow behaviour (U, needs a stateful drill)
- [x] 7 changecheck and leakcheck
