# Tasks

```yaml
change: logfields
status: archived
agent: logfields
branch: agent/logfields
worktree: ~/code/d2lod-wt/logfields
skip_specs: true
owns:
  - docs/logfields.md
  - internal/d2dsoakstat/skills.go
  - internal/d2dsoakstat/skills_test.go
  - internal/combat/castlog.go
  - internal/combat/castlog_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/combat ./internal/d2dsoakstat ./cmd/d2dsoak passes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0"
result: "offline, logging only (action-sequence test); not verified live"
live: "not verified live"
merge: f1d14cb
```

- [x] audit where each missing line can be emitted
- [x] emit lines (logging only, no behaviour change)
- [x] parse and print in d2dsoak report; update docs/soakskills.md and docs/logfields.md
