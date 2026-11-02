# Tasks

```yaml
change: kitstall2
status: archived
agent: kitstall2
branch: agent/kitstall2
worktree: d2lod-wt/kitstall2
skip_specs: true
owns:
  - internal/loot/pickup.go
  - internal/loot/loot_test.go
  - internal/town/shop.go
  - internal/town/rebuy.go
  - internal/town/earlygear_test.go
  - internal/town/sim/sim.go
  - internal/town/sim/input.go
  - openspec/changes/kitstall2/**
exclusive: []
gates:
  - "gofmt; go vet and go test of internal/loot, internal/town, internal/town/sim; changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "go tests loot town leveler pass; not verified live; staff and gloves loss cause still U"
live: "not run live"
merge: 0737a5f
```

- [x] reconstruct 11:30 to 11:40 from the lane log
- [x] loot: unwear a launcher worn by the game on a non-Amazon
- [x] town: sell refuses worn and start items, sale and worn logging, start weapon re-worn
- [x] offline tests
