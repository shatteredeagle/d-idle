# Tasks

```yaml
change: merchire
status: in-progress
agent: merchire
branch: agent/merchire
worktree: ~/code/d2lod-wt/merchire
skip_specs: true
owns:
  - docs/merchire.md
  - openspec/changes/merchire/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "D8.h and D8.h2 built with fakes and tests; lab step over d2dctl; not verified live (quote location in the menu is U)"
live: "not verified live"
```

- [x] work done (drill, lab step, tests, docs/merchire.md)
