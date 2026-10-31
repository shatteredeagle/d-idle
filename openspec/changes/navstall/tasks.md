# Tasks

```yaml
change: navstall
status: in-progress
agent: navstall
branch: agent/navstall
worktree: ~/code/d2lod-wt/navstall
owns:
  - internal/combat/fight.go
  - internal/combat/crowd.go
  - internal/combat/navstall_test.go
  - openspec/changes/navstall/**
exclusive: []
gates:
  - "go test -p 1 ./internal/combat ./internal/nav ./internal/d2dinput passes; the offline repro terminates"
  - "whole-repo go vet and go test -short pass on the worker host; go test -race ./internal/combat"
  - "soakfull seed 11 and runab seed 42 (walk-only) pass the former hang point on hidden d2d"
merge:
result: ""
```

## 1. Fix

- [x] 1.1 find the spinning loop (fight loop: Safety CrowdRetreat, failed walk, no sleep, pin hold never expires on a frozen clock)
- [x] 1.2 offline repro test, fix in move/canRetreat, watchdog, tests
- [x] 1.3 audit of other loops
- [ ] 1.4 whole-repo checks on the worker host
- [ ] 1.5 live verify both seeds on hidden d2d
