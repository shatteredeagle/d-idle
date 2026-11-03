# Tasks

```yaml
change: verifycovstates
status: in-progress
agent: verifycovstates
branch: agent/verifycovstates
worktree: ~/code/d2lod-wt/verifycovstates
skip_specs: true
owns:
  - openspec/changes/verifycovstates/**
  - docs/research/verifycovstates-trace.md
  - tools/emu/vcs_game.py
  - tools/emu/vcs_bits.py
  - tools/emu/vcs_d2d_test.cpp
  - tools/emu/vcs_run.sh
  - tools/emu/vcs_dis.sh
  - docs/coverage/ledger.tsv
  - docs/coverage/rule-coverage.md
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
  - "ledger rows only with a V run case count that appears in the trace doc"
result: "Offline, nothing verified live, no d2d slot, no Go change. Own harness: stun 25,567 + 13,029 monster cases on real MonStats rows vs the real d2d Fight::land, 0 differ; seed use D2 confirmed (2,316 of 2,316); poison 60,000 cases, deviation confirmed and larger than first reported (both directions, plus blows under one point); state bits 185 of 185; B1 and B2 confirmed. No PR candidate. Ledger rows states 21 and 2."
```

## 1. Work

- [x] 1.1 Read FUN_0057aae0, FUN_0057ac50 and the bit functions first (own disassembly)
- [x] 1.2 Own harness vcs_game.py / vcs_d2d_test.cpp, real MonStats rows, per-case seed
- [x] 1.3 Stun and poison differentials, two seeds
- [x] 1.4 State-bit probe vcs_bits.py
- [x] 1.5 Check States.txt counts and bot findings B1, B2
- [x] 1.6 Trace doc, ledger rows, `python3 tools/coverage/coverage.py --write`
- [x] 1.7 changecheck, leakcheck, merge-lines.txt
