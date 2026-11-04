# Tasks

```yaml
change: sksmoke-a1
status: archived
agent: sksmoke
branch: agent/sksmoke
worktree: ~/code/d2lod-wt/sksmoke
skip_specs: true
owns:
  - openspec/changes/sksmoke-a1/**
  - docs/probes/sksmoke/2026-11-03-SK-design.txt
  - docs/probes/sksmoke/2026-11-03-SK-d2d*
  - docs/sksmoke-tools.md
  - cmd/d2cheat/skilllevel.go
  - cmd/d2cheat/skilllevel_test.go
  - internal/combat/cmd/combatlive/sk.go
  - internal/combat/cmd/combatlive/sk_test.go
exclusive:
  - "instance:i2"
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "Fire Bolt d2d PASS (compared to table, not game.exe); Charged Bolt no verdict; game.exe live blocked"
merge: 892ed2f
```

## 1. Work

- [x] 1.1 Expected Fire Bolt and Charged Bolt damage per level 1, 4, 8, 13, 20 from skills.txt columns
- [x] 1.2 Drill design and PASS conditions
- [x] 1.5 d2cheat skilllevel (SETUP, reason, ledger)
- [x] 1.6 combatlive SK subcommand
- [x] 1.3 Fire Bolt per-hit damage at levels 1, 4, 8, 13, 20 measured on d2d (lab fork) against the table: PASS (docs/probes/sksmoke/2026-11-03-SK-d2d.txt). NOT game.exe: the i2 attempt stays BLOCKED (SK-live.txt)
- [ ] 1.4 Frost Nova ring (no harness)
