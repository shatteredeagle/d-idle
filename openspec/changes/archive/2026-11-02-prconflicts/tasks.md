# Tasks

```yaml
change: prconflicts
status: archived
agent: prconflicts
branch: agent/prconflicts
worktree: d2lod-wt/prconflicts
skip_specs: true
owns:
  - openspec/changes/prconflicts/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "log and card only"
live: "not applicable"
merge: be84737
```

- [x] read the maintainer's rules fresh
- [x] rebase the three branches, resolve conflicts
- [x] suites, differentials, parity checks
- [x] log entry
