# Tasks

```yaml
change: plandocs
status: archived
agent: pa-plandocs
branch: agent/plandocs
worktree: d2lod-wt/plandocs
skip_specs: true
owns:
  - docs/community/open-work.md
  - docs/plan/plan-v2.md
  - openspec/changes/plandocs/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; merge_lines_check --changed"
result: "docs only; not verified live"
live: "not verified"
merge: fe3b2eb
```

- [x] BACKLOG goal line and plan-v2 banner
- [x] open-work items marked done or narrowed
- [x] checks
