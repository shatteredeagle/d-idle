# Tasks

```yaml
change: fixcastgap
status: archived
agent: fixcastgap
branch: agent/fixcastgap
worktree: ~/code/d2lod-wt/fixcastgap
skip_specs: true
owns:
  - docs/fixcastgap.md
  - internal/combat/castgap.go
  - internal/combat/castgap_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/combat ./internal/env ./tools/knobdump passes including TestFightLoopLightningNeverReissuedEarly"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: on main (see result)
result: "offline only: code merged on main, live proof pending (closed 2026-11-04 by user decision). Open live proof: cast gaps per skill against the game with the lab binary carrying fix/lightning-sequence, recipe in docs/fixcastgap.md. Per-skill gap table and knobs, tests pass; not verified live"
live: "not verified live: see the live proof recipe in docs/fixcastgap.md"
```

- [x] evidence per skill and backend (docs/fixcastgap.md)
- [x] per-skill gap table, knobs castPerSkill / castMarginMs / castFCR
- [x] tests (oracle lengths, every plan skill, fallback, override, fight loop)
- [x] live proof recipe
- [ ] live proof (needs the live slot and the lab binary with fix/lightning-sequence)
