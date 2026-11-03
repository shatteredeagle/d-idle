# Tasks

```yaml
change: ccmergecommit
status: done
agent: ccmergecommit
branch: agent/ccmergecommit
worktree: d2lod-wt/ccmergecommit
skip_specs: true
owns:
  - cmd/changecheck/**
  - openspec/changes/ccmergecommit/**
exclusive: []
gates: go test -p 1 ./cmd/changecheck; merge_lines_check --changed
result: "Tooling only, not verified live. changecheck ignores the synthetic merge subject; test covers skipped and still-flagged cases."
credit:
  author: owner
  agents: [Claude Code]
  consent: none
```

- [x] 1. Skip the synthetic merge commit subject
- [x] 2. Table test
