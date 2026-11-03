# Tasks

```yaml
change: beltfix4
status: archived
agent: beltfix4
branch: agent/beltfix4
worktree: ~/code/d2lod-wt/beltfix4
skip_specs: true
owns:
  - internal/bot/goal.go
  - internal/bot/beltfix3_test.go
  - internal/bot/scenario_test.go
  - docs/research/beltfix4.md
  - openspec/changes/beltfix4/**
exclusive: []
gates:
  - "go test -p 1 -run TestRecovery ./internal/bot fails without the goal.go change and passes with it; changecheck -cli 0 warnings; leakcheck 0"
merge: 04b5fe1
result: "go tests bot pass; whole repo vet clean; scenario KitlessClvl7PackDeath changed to 29 gold"
credit:
  author: owner
  agents: [Claude Code]
```

## 1. Work

- [x] 1.1 Classify the BELT windows of gate1-20261103-0053 (docs/research/beltfix4.md)
- [x] 1.2 Fix: recovery keeps the in-town emergency potion restock (goal.go choresDue), tests TestRecoveryTownArrivalRestocksHealing and TestRecoveryStillQuietWithPotionOrNoGold
- [x] 1.3 XP/h comparison with the two earlier runs (docs/research/beltfix4.md section 3)
