# Tasks

```yaml
change: skmeleewire
status: archived
agent: skmeleewire
branch: agent/skmeleewire
worktree: ~/code/d2lod-wt/skmeleewire
skip_specs: true
owns:
  - internal/combat/melee.go
  - internal/combat/melee_test.go
  - internal/combat/resource.go
  - internal/data/items.go
  - internal/data/schema.go
  - internal/data/data_test.go
  - internal/resource/sim.go
  - internal/resource/wire_test.go
  - docs/skmeleewire.md
  - openspec/changes/skmeleewire/**
exclusive: []
gates:
  - "go test -p 1 ./internal/data ./internal/resource ./internal/combat ./cmd/d2bot; whole-repo vet and test on the worker host"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
merge: on main (see result)
result: "offline only: code merged on main, live proof pending (closed 2026-11-04 by user decision). Open live proof: task 1.5 early-game soak. Offline, not verified live: awaits an early-game soak with a staff caster and a sword fighter (drill R1 for stats 21 / 22). combat.resInput now feeds Input.Blow and Input.Foe from the snapshot and the game tables; nil keeps the old behaviour. Build-specific values all come from tables and the stat vector; no class, weapon or level literal."
live: "offline only, nothing run live"
```

## 1. Work

- [x] 1.1 weapons.txt StrBonus / DexBonus into internal/data (schema 3)
- [x] 1.2 meleeBlow / meleeFoe and the resInput feed, unit tests (staff caster, sword fighter, undead and living packs)
- [x] 1.3 fed-versus-blind replay and rung table (wire_test.go)
- [x] 1.4 docs/skmeleewire.md
- [ ] 1.5 live early-game soak (needs the worker host; not done here)
