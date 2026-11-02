# Tasks

```yaml
change: baselineruns-20261102
status: archived
agent: baselinerun
branch: agent/baselineruns-20261102
worktree: d2lod-wt/baselineruns-20261102
skip_specs: true
owns:
  - docs/probes/baseline-runs/20261102/**
  - openspec/changes/baselineruns-20261102/**
exclusive: []
gates: rows load with d2dsoak compare; weak and invalid runs flagged in record.md
result: "generated records only"
merge: d37b5b2
```

- [x] 1. Runs executed by the driver, logs pulled
- [x] 2. Rows built with d2dsoak baseline, flags in the note
- [ ] 3. Coordinator: merge rows into tools/d2dsoak/baselines*.json (changes gate numbers)
