# Tasks

```yaml
change: grillapply
status: in-progress
agent: grillapply
branch: agent/grillapply
worktree: ~/code/d2lod-wt/grillapply
owns:
  - docs/adr/0012-milestones-and-wip-limit.md
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/openspec.sh validate --all --strict passes; go build ./..."
merge:
result:
```

## 1. Work

- [x] 1.1 Merge the grill branch, ADR 0012, glossary, decisions index line
- [x] 1.2 m3-routeguard, m3-merc deltas; m3-readiness-model card; BACKLOG parked lines
- [x] 1.3 Checks
