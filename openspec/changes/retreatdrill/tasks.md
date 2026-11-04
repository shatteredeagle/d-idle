# Tasks

```yaml
change: retreatdrill
status: in-progress
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
result: "group RT (RT.drink, RT.portal, RT.leave) designed and unit-tested offline; RT.leave Blocked until Fighter.Leave is wired in d2drill; docs/retreatdrill.md has the command"
live: "not verified live"
```

- [x] work done
