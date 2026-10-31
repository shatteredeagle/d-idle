# Tasks

```yaml
change: m3-routeguard
status: proposed
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

- [ ] 1.1 Read ADR 0011 and the delta in specs/merc-policy; failing floor tests first (16/14 road and Forgotten Tower, 17/15 Andariel, antidotes in the kit, no +2 rule)
- [ ] 1.2 Knob defaults (with-merc knobs, `route.towerNeedsMerc` 0) and tests; docs/knobs.md regenerated; the Andariel attempt stays blocked until m3-andariel has won once (ADR 0011 gate 3)
- [ ] 1.3 changecheck, leakcheck, validate
