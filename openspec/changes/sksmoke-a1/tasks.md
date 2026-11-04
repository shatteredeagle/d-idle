# Tasks

```yaml
change: sksmoke-a1
status: in-progress
agent: sksmoke
branch: agent/sksmoke
worktree: ~/code/d2lod-wt/sksmoke
skip_specs: true
owns:
  - openspec/changes/sksmoke-a1/**
  - docs/probes/drills/2026-11-03-SK-design.txt
exclusive:
  - "instance:i2"
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "BLOCKED live: instances i1..i5 stopped, agents may not start games. Offline only: drill design and expected values (tables read on the worker host). Not verified live."
```

## 1. Work

- [x] 1.1 Expected Fire Bolt and Charged Bolt damage per level 1, 4, 8, 13, 20 from skills.txt columns
- [x] 1.2 Drill design and PASS conditions
- [ ] 1.3 Live run on i2 (BLOCKED: instance stopped)
- [ ] 1.4 Frost Nova ring (no harness)
