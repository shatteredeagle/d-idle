# Tasks

```yaml
change: killcredit
status: in-progress
agent: killcredit
branch: agent/killcredit
worktree: ~/code/d2lod-wt/killcredit
skip_specs: true
owns:
  - docs/killcredit.md
  - internal/d2dsoakstat/skills.go
  - internal/d2dsoakstat/skills_test.go
  - internal/combat/castlog.go
  - internal/combat/castlog_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/combat ./internal/d2dsoakstat ./cmd/d2dsoak passes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0"
result: ""
live: "not verified live"
```

- [ ] audit what the combat code knows about hits per cast and the killing skill
- [ ] emit lines (logging only, action sequence unchanged test)
- [ ] report prints both; docs
