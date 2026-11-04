# Tasks

```yaml
change: buildv3
status: archived
merge: on main (see result)
agent: buildv3
branch: agent/buildv3
worktree: ~/code/d2lod-wt/buildv3
skip_specs: true
owns:
  - internal/plandata/build-fallback-lightning-v3.json
  - internal/plandata/build-fallback-lightning-v2.json
  - docs/plan/data/build-fallback-lightning-v3.json
  - docs/plan/data/build-fallback-lightning-v2.json
  - internal/plandata/plandata_test.go
  - internal/leveler/build.go
  - internal/leveler/buildv3_test.go
  - internal/combat/classdata/profiles.json
  - internal/combat/class_test.go
  - internal/combat/pure_test.go
  - cmd/d2bot/class.go
  - cmd/d2bot/class_test.go
  - docs/buildv3.md
  - openspec/changes/buildv3/**
exclusive: []
gates:
  - "gofmt, go vet, go test -p 1 per touched package, whole repo on the worker host; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "v3 table opt-in (default stays v2), replay with and without quests refuses no row, combat band gated by learned skill; nothing run live"
live: "not run; soak recipe in docs/buildv3.md"
```

- [x] new rows and replay
- [x] combat band and selector
- [x] tests and docs
- [ ] live soak (owner after merge)
