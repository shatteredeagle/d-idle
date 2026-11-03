# Tasks

```yaml
change: branchtriage
status: archived
agent: branchtriage
branch: agent/branchtriage
worktree: d2lod-wt/branchtriage
skip_specs: true
owns:
  - docs/research/branch-triage-20261102.md
  - openspec/changes/branchtriage/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs and tools only; merged by the triage queue"
live: "docs only, nothing run live"
merge: bab1c35
```

- [x] recompute the unmerged agent branches against origin/main
- [x] per branch: unique commits, files, content on main by another route, card status, worktree state
- [x] write the triage table with delete / merge commands (not run)
