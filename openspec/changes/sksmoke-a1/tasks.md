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
  - docs/sksmoke-tools.md
  - cmd/d2cheat/skilllevel.go
  - cmd/d2cheat/skilllevel_test.go
  - cmd/combatlive/sk.go
  - cmd/combatlive/sk_test.go
exclusive:
  - "instance:i2"
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "BLOCKED live: instances i1..i5 stopped, agents may not start games. Offline only: drill design and expected values (tables read on the worker host). Not verified live. Offline tools added: d2cheat skilllevel and combatlive sk (unit tests only, docs/sksmoke-tools.md); neither was run against a game."
```

## 1. Work

- [x] 1.1 Expected Fire Bolt and Charged Bolt damage per level 1, 4, 8, 13, 20 from skills.txt columns
- [x] 1.2 Drill design and PASS conditions
- [x] 1.5 d2cheat skilllevel (SETUP, reason, ledger)
- [x] 1.6 combatlive SK subcommand
- [ ] 1.3 Live run on i2 (BLOCKED: instance stopped)
- [ ] 1.4 Frost Nova ring (no harness)
