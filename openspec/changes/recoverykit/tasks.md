# Tasks

```yaml
change: recoverykit
status: in-progress
agent: recoverykit
branch: agent/recoverykit
worktree: ~/code/d2lod-wt/recoverykit
skip_specs: true
owns:
  - openspec/changes/recoverykit/**
  - internal/resource/recovery.go
  - internal/resource/recoverykit_test.go
  - internal/bot/goal.go
exclusive: []
gates:
  - "changecheck 0 warnings for this card"
result: "Offline only, not verified live or in a gate."
```

## 1. Work

- [x] 1.1 resource.RecoveryKitTrip and table test
- [x] 1.2 Hook in bot.choresDue
- [ ] 1.3 Gate 1 rerun on a seed like 19 (another run)
