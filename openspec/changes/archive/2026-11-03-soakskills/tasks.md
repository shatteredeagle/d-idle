# Tasks

```yaml
change: soakskills
status: archived
agent: soakskills
branch: agent/soakskills
worktree: ~/code/d2lod-wt/soakskills
skip_specs: true
owns:
  - docs/soakskills.md
  - internal/d2dsoakstat/skills.go
  - internal/d2dsoakstat/skills_test.go
  - cmd/d2dsoak/main.go
  - cmd/d2dsoak/report.go
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dsoakstat ./cmd/d2dsoak passes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0"
result: "offline, fixture tests; not verified live"
live: "not verified live"
merge: 939cd91
```

- [x] audit of section 2 fields against d2dsoak report
- [x] add the missing supplied fields (two builds in tests)
- [x] docs/soakskills.md: field table, what is not logged
- [x] wire `d2dsoak report` (cmd/d2dsoak) with test
