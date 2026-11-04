# Tasks

```yaml
change: gate2run-s11-20261103-1954-g2a
status: in-progress
agent: gaterun
branch: agent/gate2run-s11-20261103-1954-g2a
worktree: /home/jesse/code/d2lod-wt/gate2run-s11-20261103-1954-g2a
skip_specs: true
owns:
  - docs/soaks/gate2-s11-20261103-1954-g2a.md
  - docs/probes/soak-logs/gate2-s11-20261103-1954-g2a-*
  - openspec/changes/gate2run-s11-20261103-1954-g2a/**
exclusive: []
gates: Gate 2 criteria of the soak-gates spec judged by d2dsoak (numbers in docs/soaks/gate2-s11-20261103-1954-g2a.md)
result: "Gate 2 NOT PASSED (Attempt Andariel at clvl 14 or higher); unattended soak by tools/gate-run.sh, run gate2-s11-20261103-1954-g2a; verified live on the worker host (script-run, not hand-run), record generated from the logs"
```

- [x] 1. Soak run unattended on a hidden d2d slot with the injected fault(s)
- [x] 2. Judge against the Gate 2 criteria, write the record
- [ ] 3. Review the record (short Sonnet agent), write fix cards on FAIL
