# Tasks

```yaml
change: gate2run-s13-20261103-2211-g2v
status: in-progress
agent: gaterun
branch: agent/gate2run-s13-20261103-2211-g2v
worktree: ~/code/d2lod-wt/gate2run-s13-20261103-2211-g2v
skip_specs: true
owns:
  - docs/soaks/gate2-s13-20261103-2211-g2v.md
  - docs/probes/soak-logs/gate2-s13-20261103-2211-g2v-*
  - openspec/changes/gate2run-s13-20261103-2211-g2v/**
exclusive: []
gates: Gate 2 criteria of the soak-gates spec judged by d2dsoak (numbers in docs/soaks/gate2-s13-20261103-2211-g2v.md)
result: "Gate 2 NOT PASSED (XP/h at or above the m24 floor (1000 per hour); No single state over 10 min without progress; Belt; d2dsoak INVARIANTS; No-gain window at most 90 min (clvl or gold gain; quest step not read; Attempt Andariel at clvl 14 or higher; Hire the merc); unattended soak by tools/gate-run.sh, run gate2-s13-20261103-2211-g2v; verified live on the worker host (script-run, not hand-run), record generated from the logs"
```

- [x] 1. Soak run unattended on a hidden d2d slot with the injected fault(s)
- [x] 2. Judge against the Gate 2 criteria, write the record
- [ ] 3. Review the record (short Sonnet agent), write fix cards on FAIL
