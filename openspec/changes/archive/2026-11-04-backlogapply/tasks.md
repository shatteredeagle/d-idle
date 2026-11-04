# Tasks

```yaml
change: backlogapply
status: archived
agent: backlogapply
branch: agent/backlogapply
worktree: ~/code/d2lod-wt/backlogapply
skip_specs: true
owns:
  - docs/research/backlog-apply-20261104.md
  - openspec/changes/backlogapply/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits; merge_lines_check ok"
result: "docs only; offline-only stays offline-only in the new lines"
live: "not verified live (docs only)"
merge: a81936d
```

- [ ] work done
