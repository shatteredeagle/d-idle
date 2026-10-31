# Tasks

```yaml
change: goldfloor
status: in-progress
agent: goldfloor
branch: agent/goldfloor
worktree: ~/code/d2lod-wt/goldfloor
owns:
  - internal/bot/goal.go
  - internal/bot/goldfloor_test.go
  - internal/town/shop.go
  - internal/town/beltbuy.go
  - internal/town/goldfloor_test.go
  - internal/env/tuning_town.go
  - internal/env/tuning_loot.go
  - docs/goldfloor.md
exclusive: []
gates:
  - "go test -p 1 ./internal/bot ./internal/town ./internal/env passes including TestTripFloor (a [2 healing potions] trip with a 30-gold plan and a held scroll is not raised, a belt at 0 healing is), TestNoTPRebuyWhenPoor (gold 209, 1 healing potion: one potion plan, no scroll), TestManaCapBelowClvl10"
  - "paired A/B 8 seeds x 2 virtual h per arm, same binary: trips under 30 gold at most 35 percent (was 65 to 83), TP spend at most 250 gold/h in bands 5-10 (was 156 to 760), healing belt empty share not above baseline, deaths not above baseline"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result: "1.1 and 1.2 implemented offline (d2d-only: trip floor, shop and mana cap are the town module used by the d2d backend); unit and scenario tests pass; the paired 8-seed A/B gate (1.3) is NOT run, the loot bar trial (lever C) waits for it; extra files touched outside owns: internal/town/chores.go (mana trip gate), internal/town/basicsfix.go (poor scroll rule)"
```

## 1. Work

- [x] 1.1 Trip floor and no TP rebuy when poor in the trip trigger and the shop plan; the three tests; knobs `town.tripMinSpend` (60) in docs/knobs.md; planned spend = `Town.PlannedSpend` (belt plan shortfall, mana capped, first scroll, at most the gold); the floor skips survival trips (free heal), sell, pack, merc and non-restock reasons
- [x] 1.2 Mana cap below clvl 10 (knob `town.mpCapEarly`, 1) and drop the mana-belt trip trigger below clvl 10; test
- [ ] 1.3 A/B on the worker host per gates; if trips and TP spend fall without more deaths, then trial `loot.minValuePerCell` 20 as a second arm; fill result
