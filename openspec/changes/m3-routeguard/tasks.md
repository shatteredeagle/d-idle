# Tasks

```yaml
change: m3-routeguard
status: deferred
agent: m3-routeguard
branch: agent/m3-routeguard
worktree: ~/code/d2lod-wt/m3-routeguard
owns:
  - internal/env/tuning_route.go
  - internal/env/tuning_boss.go
  - internal/bot/routeguard.go
  - internal/bot/routeguard_test.go
  - docs/knobs.md
exclusive: []
gates:
  - "knob defaults equal the decided numbers (docs/knobs.md regenerated; go test ./internal/env -run EveryKnob passes)"
  - "go test -p 1 ./internal/bot ./internal/task passes including TestPickNeverEmptyInAct1 and the route guard tests at the new floors"
  - "merc-policy MODIFIED blocks for Route guard and Andariel floor are complete (all scenarios) and validate --strict passes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Take the decided numbers from docs/decisions.md (grill 0003); write the MODIFIED deltas first (complete requirements, no Previously notes)
- [ ] 1.2 Knob defaults and tests; docs/knobs.md regenerated
- [ ] 1.3 changecheck, leakcheck, validate
