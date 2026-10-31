# Tasks

```yaml
change: m1-trips
status: deferred
agent: m1-trips
branch: agent/m1-trips
worktree: ~/code/d2lod-wt/m1-trips
owns:
  - internal/bot/goal.go
  - internal/bot/trips_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/bot passes including TestChoresDueSecondDoor (4 or 5 sellable items with a scroll and gold under 150 raise no trip), TestGoldTripAboveKeep (carried gold 400 above the keep line raises reason `gold`), TestPackFullFreeCells (4 free cells raise one sell trip, 5 do not)"
  - "go test ./internal/env -run EveryKnob passes with the new knob and docs/knobs.md regenerated"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Re-read docs/basics-audit.md rows 4, 14, E6 and G2, G8, G9 against the merged tree (xpgap owns goal.go today: its povertySellMin knob may already close G2; strike it in the launch brief if so)
- [ ] 1.2 Second door uses povertySellWorth; TestChoresDueSecondDoor
- [ ] 1.3 Gold trip reason above keep + margin; TestGoldTripAboveKeep
- [ ] 1.4 One pack-full knob in free cells; TestPackFullFreeCells; docs/knobs.md regenerated (owned by xpgap until archived: say so in the report)
- [ ] 1.5 changecheck, leakcheck, validate
