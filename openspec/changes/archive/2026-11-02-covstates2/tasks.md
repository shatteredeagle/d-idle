# Tasks

```yaml
change: covstates2
status: archived
agent: covstates2
branch: agent/covstates2
worktree: d2lod-wt/covstates2
skip_specs: true
owns:
  - docs/research/covstates2-trace.md
  - tools/emu/cs2_*
  - openspec/changes/covstates2/**
exclusive: []
gates:
  - game.exe FUN_0056e970 and FUN_0056c740 run natively with counted cases, compared with d2d's real put_state
  - ledger rows only for states proven, regenerated with tools/coverage/coverage.py --write
result: "docs/tools only; see the doc for tags and limits"
credit:
  component: COVSTATES2
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
merge: ea2ff01
```

- [x] 1 Read the state table (States.txt, patched stack), the curse put FUN_0056e970, the group removal FUN_0056c740, the curse do FUN_005c37c0 / FUN_005c35c0, Taunt and Battle Cry
- [x] 2 Harness tools/emu/cs2_put.py (game side), cs2_d2d_test.cpp + cs2_build.sh (d2d put_state), cs2_run.sh (end to end), cs2_len.py (lengths)
- [x] 3 Differential run: 100,000 put sequences (seed 20261102), group, gate and curse-resistance modes
- [x] 4 Trace doc docs/research/covstates2-trace.md
- [x] 5 Ledger rows for the proven states, coverage table regenerated
- [ ] 6 PR candidate (none: the deviations are rules d2d lacks, a fix needs a design decision; see the doc)
- [x] 7 changecheck and leakcheck
