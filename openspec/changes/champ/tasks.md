# Tasks

```yaml
change: champ
status: in-progress
agent: champ
branch: agent/champ
worktree: ~/code/d2lod-wt/champ
owns:
  - openspec/changes/champ/**
  - docs/champions.md
  - docs/knobs.md
  - internal/combat/target.go
  - internal/combat/death.go
  - internal/combat/resource.go
  - internal/combat/champion_test.go
  - internal/resource/potion.go
  - internal/resource/tuning.go
  - internal/resource/types.go
  - internal/resource/policy_test.go
  - internal/env/tuning_combat.go
  - internal/bot/observe.go
  - internal/bot/deathlog.go
  - internal/bot/death.go
  - internal/bot/champion_test.go
exclusive: []
gates:
  - "go vet ./..., go test -short -count=1 -p 4 ./... and go test -race ./internal/combat ./internal/bot ./internal/d2 pass on the worker host"
  - "changecheck, leakcheck -mode fail, tools/openspec.sh validate --all --strict pass"
merge: ""
result: ""
```

- [x] 1 Read what Boss gates (docs/champions.md)
- [x] 2 Champion flag, group avoid, knob, autopsy label, danger-table guard
- [x] 3 Tests
- [ ] 4 G1/G2 check (offline only here, not verified live)
