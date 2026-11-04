# Tasks

```yaml
change: primadd
status: in-progress
agent: primadd
branch: agent/primadd
worktree: ~/code/d2lod-wt/primadd
skip_specs: true
owns:
  - docs/research/primitives-additions-20261104.md
  - openspec/changes/primadd/**
exclusive: []
gates:
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; python3 tools/merge_lines_check.py --changed ok"
result: "docs/research/primitives-additions-20261104.md: draft rows P37 to P45 (Teleport movement, buff upkeep, object interaction, difficulty change, charms, threshold gear swap, chicken exit, weapon swap, merc gear equip), priority order and missing d2d verbs; docs/primitives.md untouched; not verified live"
live: "not verified live (docs only)"
```

- [x] work done
