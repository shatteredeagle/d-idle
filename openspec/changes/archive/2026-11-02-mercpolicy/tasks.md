# Tasks

```yaml
change: mercpolicy
status: archived
agent: mercpolicy
branch: agent/mercpolicy
worktree: d2lod-wt/mercpolicy
owns:
  - openspec/changes/mercpolicy/**
  - internal/merc/**
  - internal/env/tuning_merc.go
  - internal/env/tuning_route.go
  - internal/bot/routeguard.go
  - internal/bot/routeguard_test.go
  - internal/task/floors_test.go
  - cmd/d2bot/mercwire.go
  - cmd/d2bot/mercwire_test.go
  - cmd/d2bot/d2dbackend_test.go
  - docs/merc-wire.md
  - docs/knobs.md
  - docs/botlogs.md
exclusive: []
gates:
  - "go test -p 1 ./internal/merc/... ./internal/env ./internal/task ./internal/bot ./tools/knobdump ./cmd/d2bot passes"
  - "revive when affordable, not when poor, kit reserve, cap 2 an hour, Act 2 hire with strip first, hire window line, floors with and without merc: named tests in cmd/d2bot and internal/merc"
  - "changecheck 0 warnings; leakcheck 0 hits"
result: "whole-repo go test -short passes on the worker host; not verified live"
live: "not run"
credit:
  component: mercpolicy
  author: owner
  agents: [Claude Code]
  consent: none
merge: fdbc334
```

- [x] Knobs: merc.enabled 1, kitReservePct, goldFarmMaxLevel, reviveGoldMult 1, hireGoldMult 1, potionLifePct 50, monasteryMinLevel 14; docs/knobs.md regenerated
- [x] internal/merc: kit reserve in revive and hire, KitRefill, GoldFarm, FarmAreas, wait and leave helpers, log lines
- [x] cmd/d2bot: HireStep (strip first, hire window line), revive estimate vs paid, GoldFarm hook, bot.MercGoldFarm in guardedAreas
- [x] Tests: revive affordable and poor, reserve, cap, hire strip order, floors with and without merc
- [x] Delta spec merc-policy, docs/botlogs.md, docs/merc-wire.md
- [ ] Live: revive price read at Kashya, hire window read at Greiz, d2d Hirer implementation (needs the user)
