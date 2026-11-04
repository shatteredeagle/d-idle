# Tasks

```yaml
change: gate1run-20261101
status: archived
merge: records on main (docs/soaks/gate1-20261101.md)
agent: gaterun
branch: agent/gate1run-20261101
worktree: ~/code/d2lod-wt/gate1run-20261101
skip_specs: true
owns:
  - docs/soaks/gate1-20261101.md
  - docs/probes/soak-logs/gate1-20261101-*
  - openspec/changes/gate1run-20261101/**
exclusive: []
gates: Gate 1 criteria of the soak-gates spec judged by d2dsoak (numbers in docs/soaks/gate1-20261101.md)
result: "Gate 1 NOT PASSED (True XP/h at or above the d2d baseline; No single state over 10 min without progress); unattended soak by tools/gate-run.sh, run gate1-20261101-1704; verified live on the worker host (script-run, not hand-run), record generated from the logs"
```

- [x] 1. Soak run unattended on a hidden d2d slot with the injected fault(s)
- [x] 2. Judge against the Gate 1 criteria, write the record
- [ ] 3. Review the record (short Sonnet agent), write fix cards on FAIL
