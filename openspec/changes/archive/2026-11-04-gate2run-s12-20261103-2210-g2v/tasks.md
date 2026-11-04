# Tasks

```yaml
change: gate2run-s12-20261103-2210-g2v
status: archived
agent: gaterun
branch: agent/gate2run-s12-20261103-2210-g2v
worktree: ~/code/d2lod-wt/gate2run-s12-20261103-2210-g2v
skip_specs: true
owns:
  - docs/soaks/gate2-s12-20261103-2210-g2v.md
  - docs/probes/soak-logs/gate2-s12-20261103-2210-g2v-*
  - openspec/changes/gate2run-s12-20261103-2210-g2v/**
exclusive: []
gates: Gate 2 criteria of the soak-gates spec judged by d2dsoak (numbers in docs/soaks/gate2-s12-20261103-2210-g2v.md)
result: "Gate 2 NOT PASSED (Attempt Andariel at clvl 14 or higher); unattended soak by tools/gate-run.sh, run gate2-s12-20261103-2210-g2v; verified live on the worker host (script-run, not hand-run), record generated from the logs"
merge: 1fca126
```

- [x] 1. Soak run unattended on a hidden d2d slot with the injected fault(s)
- [x] 2. Judge against the Gate 2 criteria, write the record
- [ ] 3. Review the record (short Sonnet agent), write fix cards on FAIL
