# Tasks

```yaml
change: crlimits
status: archived
agent: crlimits
branch: agent/crlimits
worktree: d2lod-wt/crlimits
skip_specs: true
owns:
  - docs/research/coderabbit-limits-20261102.md
  - openspec/changes/crlimits/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits"
result: "docs only"
live: "not applicable (docs only)"
merge: 00058c9
```

- [x] gather limits (repo config, upstream PR history, CodeRabbit docs)
- [x] write research doc with recommendation
