# Tasks

```yaml
change: beltfix3
status: archived
agent: beltfix3
branch: agent/beltfix3
worktree: ~/code/d2lod-wt/beltfix3
skip_specs: true
owns:
  - internal/town/trip.go
  - internal/town/beltfix3_test.go
  - internal/bot/goal.go
  - internal/bot/beltfix3_test.go
  - internal/bot/beltfix_test.go
  - internal/resource/town.go
  - internal/resource/policy_test.go
  - internal/combat/resource_test.go
  - cmd/d2bot/d2dwire.go
  - cmd/d2bot/d2dportal_test.go
  - docs/research/beltfix3.md
  - openspec/changes/beltfix3/**
exclusive: []
gates:
  - "go test -p 1 ./internal/bot ./internal/town ./internal/resource ./internal/combat ./cmd/d2bot; whole repo vet+test on the worker host; changecheck -cli 0 warnings; leakcheck -mode fail 0"
merge: 7a52b7e
result: "whole repo vet and test pass on the worker host; replay test of minute lines 218-254"
live: "not verified; Gate 1 re-run checklist in docs/research/beltfix3.md"
credit:
  author: owner
  agents: [Claude Code]
```

- [x] 1 shop on every town arrival without healing potions (goal.go; TestTownArrivalRestocksWithoutCooldown)
- [x] 2 level-chain walk home and wider walkableHome (trip.go, resource/town.go; TestToTownWalksChainFromDarkWood)
- [x] 3 cut-cast error and reuse of the cut cast's portal (d2dwire.go; TestPortalCutAfterClickReportsSpent)
- [x] 4 BackToField retries the portal once (TestBackToFieldRetriesPortalOnce)
- [x] replay test of the Gate 1 window (TestReplayGate1BeltWindowDecisions)
- [ ] 5 and 6 of the analysis are NOT done (see docs/research/beltfix3.md)
