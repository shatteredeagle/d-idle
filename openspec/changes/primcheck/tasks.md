# Tasks

```yaml
change: primcheck
status: in-progress
agent: primcheck
branch: agent/primcheck
worktree: ~/code/d2lod-wt/primcheck
skip_specs: true
owns:
  - docs/research/primitives-open-20261104.md
  - openspec/changes/primcheck/**
exclusive: []
gates:
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; python3 tools/merge_lines_check.py --changed ok"
result: ""
live: "not verified live (docs only)"
```

- [ ] work done
