# Tasks

```yaml
change: m1-loot
status: proposed
agent: m1-loot
branch: agent/m1-loot
worktree: ~/code/d2lod-wt/m1-loot
owns:
  - internal/loot/pickit_starter.json
  - internal/loot/charms_test.go
  - docs/loot-charms.md
exclusive: []
gates:
  - "go test -p 1 ./internal/loot passes including TestCharmsRunesGemsOnSellVisit (every rune, every identified charm with a kept stat and every gem the rule keeps survive; the plain charm is sold only after identification)"
  - "go test -p 1 ./internal/town -run TestAuditS6 still passes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/basics-audit.md row 6 and gap G7, the pickit rules in internal/loot/rules.go and pickit_starter.json; write the failing test first
- [ ] 1.2 Pickit rule change for charms, runes, low gems per docs/roadmap.md decision 6 (default SME rule)
- [ ] 1.3 docs/loot-charms.md; changecheck, leakcheck, validate
