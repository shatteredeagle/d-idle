# Tasks

```yaml
change: hirefix
status: archived
agent: hirefix
branch: agent/hirefix
worktree: ~/code/d2lod-wt/hirefix
skip_specs: true
owns:
  - internal/d2dsrc/player.go
  - internal/d2dsrc/tables.go
  - internal/d2dmerc/**
  - cmd/d2bot/mercwire.go
  - docs/research/hirefix.md
  - openspec/changes/hirefix/**
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dmerc ./internal/d2dsrc ./cmd/d2bot; changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "go tests on touched packages pass; whole-repo checks run by merge.sh"
live: "not verified; drill recipe items 1, 2, 3 in docs/research/hirereader-trace.md"
credit:
  author: owner
  agents: [Claude Code]
merge: 94b7a62
```

- [x] B2 dead and gone
- [x] B1 codes
- [x] B4 quote
- [ ] B3 Hirer (deferred, see docs/research/hirefix.md)
- [x] judge check
