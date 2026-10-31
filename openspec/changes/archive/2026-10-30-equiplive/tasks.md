# Tasks

```yaml
change: equiplive
status: archived
agent: equiplive
branch: agent/equiplive
worktree: ~/code/d2lod-wt/equiplive
owns:
  - internal/town/rebuy.go
  - internal/town/hand/**
  - docs/gear-gate.md
  - internal/town/earlygear_test.go
  - internal/town/refused_test.go
exclusive: []
gates:
  - "no live input; screenshots of the i5 window only"
merge: 5d2a999
result: "diagnosis: items refused by the game (red tiles, likely unmet affix requirements), root cause unproven (S, 65%); fix: per-click log lines and skip-until-level-up; belt Free() overcount in beltfill.go not fixed; live effect pending"
```
