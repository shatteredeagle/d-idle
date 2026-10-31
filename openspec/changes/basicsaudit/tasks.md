# Tasks

```yaml
change: basicsaudit
status: in-progress
agent: basicsaudit
branch: agent/basicsaudit
worktree: ~/code/d2lod-wt/basicsaudit
owns:
  - docs/basics-audit.md
  - internal/town/basicsaudit_test.go
  - openspec/changes/basicsaudit/**
exclusive: []
gates:
  - "go test -p 1 ./internal/town passes on the worker host"
  - "go run ./cmd/changecheck -cli has 0 warnings; leakcheck -mode fail finds 0"
merge:
result:
```

## 1. Work

- [x] 1.1 audit the 15 SME scenarios and the extra basics against code, tests and d2d docs
- [x] 1.2 cross-check the three quiet leaks against docs/d2d-economy.md
- [x] 1.3 add offline scenario tests where the sim rig already supports them
- [x] 1.4 write docs/basics-audit.md
