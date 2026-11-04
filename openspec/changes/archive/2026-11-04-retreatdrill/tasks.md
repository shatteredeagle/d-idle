# Tasks

```yaml
change: retreatdrill
status: archived
agent: retreatdrill
branch: agent/retreatdrill
worktree: ~/code/d2lod-wt/retreatdrill
skip_specs: true
owns:
  - docs/retreatdrill.md
  - openspec/changes/retreatdrill/**
exclusive: []
gates:
  - "go test -p 1 per touched package passes; changecheck 0 warnings; leakcheck 0 hits"
result: "RT.drink and RT.portal ready for a live run; RT.leave blocked until Leave is wired"
live: "not verified live"
merge: 789ba3f
```

- [x] work done
