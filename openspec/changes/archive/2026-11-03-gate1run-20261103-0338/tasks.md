# Tasks

```yaml
change: gate1run-20261103-0338
status: archived
agent: gaterun
branch: agent/gate1run-20261103-0338
worktree: <worktree>
skip_specs: true
owns:
  - docs/soaks/gate1-20261103-0338.md
  - docs/probes/soak-logs/gate1-20261103-0338-*
  - openspec/changes/gate1run-20261103-0338/**
exclusive: []
gates: Gate 1 criteria of the soak-gates spec judged by d2dsoak (numbers in docs/soaks/gate1-20261103-0338.md)
result: "docs and tools only; see card result"
merge: bdc81dc
```

- [x] 1. Soak run unattended on a hidden d2d slot with the injected fault(s)
- [x] 2. Judge against the Gate 1 criteria, write the record
- [ ] 3. Review the record (short Sonnet agent), write fix cards on FAIL
