# Tasks

```yaml
change: lockstepbase
status: archived
agent: lockstepbase
branch: agent/lockstepbase
worktree: d2lod-wt/lockstepbase
skip_specs: true
owns:
  - docs/process/baseline-runs.md
  - internal/gclock/gclock.go
  - internal/gclock/gclock_test.go
  - cmd/d2bot/main.go
  - openspec/changes/lockstepbase/**
exclusive: []
gates:
  - "go test -p 1 ./internal/gclock; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh"
result: "tests offline; live run pending"
merge: 30c7c6c
```

- [x] 1. gclock.StartNote and the d2bot start line (K != 1 only), test
- [x] 2. runbook section 10 (hooked lanes, game time, regime, watchdog)
- [x] 3. ops repo: baseline-run.sh --timescale, hook-inst.sh, hookwatch.py, live.sh env forwarding, tests (committed there)
- [ ] 4. first live use: set kitk2 (coordinator, after set kit ends)
