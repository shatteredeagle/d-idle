# Tasks

```yaml
change: fixmanamodel
status: archived
agent: fixmanamodel
branch: agent/fixmanamodel
worktree: ~/code/d2lod-wt/fixmanamodel
skip_specs: true
owns:
  - docs/fixmanamodel.md
  - internal/resource/manamodel.go
  - internal/resource/manamodel_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/resource ./internal/combat passes; whole-repo go vet and go test -short on the worker host pass"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: on main (see result)
result: "offline only: code merged on main, live proof pending (closed 2026-11-04 by user decision). Open live proof: modelled mana against the dump over a fight, see docs/fixmanamodel.md. Regen and booking models fixed, tests pass; not verified live"
live: "not verified live: see the proof idea in docs/fixmanamodel.md"
```

- [x] regen model (per-frame whole 256ths)
- [x] tracker booking per backend
- [x] potion over time in the sim
- [x] tests per backend
- [ ] live proof (modelled mana vs the dump over a fight)
