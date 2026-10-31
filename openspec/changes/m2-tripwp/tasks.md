# Tasks

```yaml
change: m2-tripwp
status: deferred
agent: m2-tripwp
branch: agent/m2-tripwp
worktree: ~/code/d2lod-wt/m2-tripwp
owns:
  - internal/town/trip.go
  - internal/town/trip_test.go
  - internal/bot/waypoints.go
  - internal/bot/waypoints_test.go
  - docs/tripwp.md
  - docs/probes/tripwp/**
exclusive:
  - soak-slot:3
gates:
  - "paired before/after per docs/ab-design.md: `Returning to the field` at most 7 percent of time (was 14) and bot restarts per virtual hour at most half of before, interval excluding zero, raw in docs/probes/tripwp"
  - "go test -p 1 ./internal/town ./internal/bot -run 'Trip|Waypoint|Chicken' passes including TestLeaveTownByActiveWaypoint and TestChickenPortalsWhenScrollHeld"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m2-tripwp leaves nothing"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/d2d-xp-gap.md (trip costs), internal/town/trip.go, internal/bot/waypoints.go (the 10 minute failed-trip latch), docs/basics-audit.md E3; failing tests first
- [ ] 1.2 Waypoint out of town and portal return; chicken by portal when a scroll is held
- [ ] 1.3 Paired before/after; docs/tripwp.md; changecheck, leakcheck, validate; reap
