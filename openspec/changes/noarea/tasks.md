# Tasks

```yaml
change: noarea
status: in-progress
agent: noarea
branch: agent/noarea
worktree: ~/code/d2lod-wt/noarea
skip_specs: true
owns:
  - docs/noarea-policy.md
  - openspec/changes/noarea/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits; merge_lines_check ok"
result: "docs/noarea-policy.md: picker rules, seed 13 cause from log lines, seeds 11/12 comparison, 7 options with recommendation; no code; not verified live"
live: "not verified live (docs only)"
```

- [x] work done
