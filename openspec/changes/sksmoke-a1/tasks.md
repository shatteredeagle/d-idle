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
  - internal/combat/cmd/combatlive/sk.go
  - internal/combat/cmd/combatlive/sk_test.go
exclusive:
  - "instance:i2"
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "Live attempt on i2 (Combi lvl 30, Fire Bolt hard level 3, no table row): BLOCKED, method invalid. Verified live: stat 6 of a client monster is a 0..128 percent scale (full = 128), so life deltas are not damage points; table levels 1/4/8/13/20 not run (skill levels need the game stopped). Evidence docs/probes/drills/2026-11-03-SK-live.txt. Charged Bolt, Frost Nova, pack drill not done. Offline follow-up: no exact monster life read exists on game.exe (d2d monsters hp is exact but d2d only), so combatlive sk now measures bolts to kill with monstats max life (an estimate bracket, not per-hit points, per-cast stderr progress); tests with the fake reader pass; NOT run live."
```

## 1. Work

- [x] 1.1 Expected Fire Bolt and Charged Bolt damage per level 1, 4, 8, 13, 20 from skills.txt columns
- [x] 1.2 Drill design and PASS conditions
- [x] 1.5 d2cheat skilllevel (SETUP, reason, ledger)
- [x] 1.6 combatlive SK subcommand
- [ ] 1.3 Live run on i2 (attempted at level 3: method invalid, see SK-live.txt)
- [ ] 1.4 Frost Nova ring (no harness)
