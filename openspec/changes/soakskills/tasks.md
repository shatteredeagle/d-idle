# Tasks

```yaml
change: soakskills
status: in-progress
agent: soakskills
branch: agent/soakskills
worktree: ~/code/d2lod-wt/soakskills
skip_specs: true
owns:
  - docs/soakskills.md
  - internal/d2dsoakstat/skills.go
  - internal/d2dsoakstat/skills_test.go
  - cmd/d2dsoak/main.go
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dsoakstat ./cmd/d2dsoak passes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0"
result: "skills.go prints casts, buffs, recasts, novas, swings, casts/kill, build; rest n/a, see docs/soakskills.md; cmd/d2dsoak hook pending"
live: "not verified live"
```

- [x] audit of section 2 fields against d2dsoak report
- [x] add the missing supplied fields (two builds in tests)
- [x] docs/soakskills.md: field table, what is not logged
