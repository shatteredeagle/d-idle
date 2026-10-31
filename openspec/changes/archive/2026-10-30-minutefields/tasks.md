# Tasks

```yaml
change: minutefields
status: archived
agent: minutefields
branch: agent/minutefields
worktree: ~/code/d2lod-wt/minutefields
owns:
  - internal/bot/hpfields.go
  - internal/bot/observe.go
  - internal/bot/m0logs_test.go
exclusive: []
gates:
  - "go test -p 1 ./internal/bot passes; field names match internal/d2dsoakstat/minute.go (gatedelta)"
merge:
result: "offline only (fake-state unit test); not verified live"
merge: 9d36c38
result: "minute line emits dmgTaken (running raw HP lost) and hpMinPct; not verified live"
```

## 1. Work

- [x] 1.1 emit dmgTaken and hpMinPct in minuteLine
- [x] 1.2 unit test
