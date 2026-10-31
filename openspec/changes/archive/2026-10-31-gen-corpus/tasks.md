# Tasks

```yaml
change: gen-corpus
status: archived
agent: gencorpus
branch: agent/gencorpus
worktree: ~/code/d2lod-wt/gencorpus
skip_specs: true
owns:
  - internal/nav/scenario_test.go
  - internal/bot/scenario_test.go
  - internal/town/scenario_test.go
  - internal/combat/scenario_test.go
  - docs/scenarios.md
exclusive: []
gates:
  - "go test -short -count=1 ./internal/nav ./internal/bot ./internal/town ./internal/combat -run TestScenario passes, each knob-off subtest reproduces the old failure"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: 933c213
result: "6 scenarios as TestScenario_* tests + docs/scenarios.md; offline only, not verified live; knob-off subtests reproduce the old failures"
```

## 1. Work

- [x] 1.1 TestScenario_KitlessClvl7PackDeath (bot)
- [x] 1.2 TestScenario_WedgePocket (nav)
- [x] 1.3 TestScenario_PovertyLoop (town + bot)
- [x] 1.4 TestScenario_CampaignStall (bot)
- [x] 1.5 optional: combat pin and champion group scenarios
- [x] 1.6 docs/scenarios.md index; changecheck, leakcheck, validate
