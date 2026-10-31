# Tasks

```yaml
change: gearsetup
status: archived
agent: gearsetup
branch: agent/gearsetup
worktree: ~/code/d2lod-wt/gearsetup
owns:
  - internal/gear/**
  - cmd/d2tables/**
  - THIRD_PARTY.md
  - docs/quickstart.md
  - docs/gear.md
exclusive: []
gates:
  - "extractor output equals the old committed JSON on the real data (TestMatchesLegacyJSON on the worker host)"
  - "go test ./internal/gear ./cmd/d2tables passes without any game data"
merge: 8e42f26
result: "Blizzard gear tables are no longer committed: cmd/d2tables extracts them at setup from the user own game files into ~/.cache/d2lod/tables; value-level equality with the old JSON (98 types, 362 bases, 210 skills, 7 classes, 202 armor rows); ~80 tests skip without tables"
```

## 1. Work

- [x] 1.1 Go extractor + cmd/d2tables + stamp + cache loader
- [x] 1.2 equality vs old JSON on the worker host (types 98, bases 362, skills 210, classes 7, armor 202: equal)
- [x] 1.3 synthetic fixture tests; real-value tests skip without tables
- [x] 1.4 remove committed JSON; THIRD_PARTY, quickstart, gear.md, .gitignore
- [x] 1.5 whole-repo vet and `go test -short ./...` on the worker host: clean with D2_DATA_DIR=/tmp/d2mpq (first-run auto-extract) and clean with an empty cache (real-value tests skip)
