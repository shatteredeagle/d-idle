# Tasks

```yaml
change: fixskilllvl
status: archived
agent: fixskilllvl
branch: agent/fixskilllvl
worktree: ~/code/d2lod-wt/fixskilllvl
skip_specs: true
owns:
  - docs/fixskilllvl.md
  - internal/d2/skilllevel.go
  - internal/d2/skilllevel_test.go
  - internal/combat/skilllevel_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/d2 ./internal/combat passes; whole-repo go vet and go test -short on the worker host pass"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: on main (see result)
result: "offline only: code merged on main, live proof pending (closed 2026-11-04 by user decision). Open live proof: stat vector with +skills gear and drills P1-P6 of the SKPASS trace, see docs/fixskilllvl.md. Bonus rules implemented and tested; not verified live"
live: "not verified live: the stat ids and sub indexes on the real stat vector, see docs/fixskilllvl.md"
```

- [x] bonus rule table (127, shrine, 83, 188, 97 cap, 126, 107)
- [x] KnownSkills casts at the effective level
- [x] tests per source and combination, base-only regression
- [ ] live proof (stat vector with +skills gear, drills P1-P6 of the SKPASS trace)
