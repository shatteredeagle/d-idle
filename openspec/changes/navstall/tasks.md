# Tasks

```yaml
change: navstall
status: done
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
result: "verified live on hidden d2d (lab6, lockstep, 2 virtual hours each): soakfull seed 11 (clvl 5, 354 kills, 9742 xp, 0 deaths) and runab seed 42 walk-only (clvl 5, 381 kills, 9075 xp, 0 deaths) pass the former hang points and finish; the watchdog never fired (the failed-walk tick was enough); vet, short tests and race for internal/combat pass on the worker host; Game.exe-affecting (shared combat code): needs a G1/G2 check"
```

## 1. Fix

- [x] 1.1 find the spinning loop (fight loop: Safety CrowdRetreat, failed walk, no sleep, pin hold never expires on a frozen clock)
- [x] 1.2 offline repro test, fix in move/canRetreat, watchdog, tests
- [x] 1.3 audit of other loops
- [x] 1.4 whole-repo checks on the worker host
- [x] 1.5 live verify both seeds on hidden d2d
