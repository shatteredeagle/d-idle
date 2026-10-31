# Tasks

```yaml
change: kitgate
status: in-progress
agent: kitgate
branch: agent/kitgate
worktree: ~/code/d2lod-wt/kitgate
skip_specs: false
owns:
  - internal/bot/kitgate.go
  - internal/bot/kitgate_test.go
  - internal/bot/broke.go
  - internal/bot/goal.go
  - internal/bot/recoveryxp_test.go
  - internal/bot/leveling_test.go
  - cmd/d2bot/resource_test.go
  - internal/nav/nav.go
  - internal/nav/route.go
  - internal/nav/walk.go
  - internal/nav/exit_test.go
  - internal/env/tuning_resource.go
  - internal/resource/tuning.go
  - docs/knobs.md
  - docs/kitgate.md
exclusive: []
gates:
  - "go test -p 1 ./internal/bot -run 'KitGate|Recovery' and ./internal/nav -run RouteAvoid pass"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [x] 1.1 Read the recovery code path, write docs/kitgate.md
- [x] 1.2 Kit gate in recoveryArea and target0, knob recoveryKitGate (default 1)
- [x] 1.3 nav.SetAvoidLevels, Route cost, NearMouth
- [x] 1.4 Table tests, regression test, spec delta, docs/knobs.md
