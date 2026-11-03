# Tasks

```yaml
change: mercwire
status: archived
agent: mercwire
branch: agent/mercwire
worktree: d2lod-wt/mercwire
owns:
  - openspec/changes/mercwire/**
  - internal/combat/mercfield.go
  - internal/combat/mercfield_test.go
  - internal/bot/mercwait.go
  - internal/bot/mercwait_test.go
  - cmd/d2bot/mercwire.go
  - cmd/d2bot/mercwire_test.go
  - docs/merc-wire.md
  - docs/botlogs.md
exclusive: []
gates:
  - "go test -p 1 ./internal/combat ./internal/bot ./internal/merc/... ./cmd/d2bot passes"
  - "wait ends on arrival and is bounded, leave-with-merc-down respects the runner guard, potion given once per gap and logs (no verb), first-visit flag survives a restart: named tests in internal/combat, internal/bot, cmd/d2bot"
  - "changecheck 0 warnings; leakcheck 0 hits"
result: "go tests pass; potion verb effect and live behaviour unverified"
live: "not run"
credit:
  component: mercwire
  author: owner
  agents: [Claude Code]
  consent: none
merge: 597e972
```

- [x] Wait for the Rogue after a portal or waypoint (bot.MercWait, mercWire.WaitForMerc, bounded 6 s, log lines)
- [x] Leave when the crowd rule fires with her down (Fighter.mercDownLeave, runner guard respected, once-a-minute log)
- [x] Healing potion for the merc below 50 percent (Fighter.mercPotion over the Belt(col, merc) verb; no-verb input logs once a minute)
- [x] First-visit flag persisted per character and difficulty (instance state merc.json)
- [x] Tests for each, docs/botlogs.md, docs/merc-wire.md
- [ ] Live: cmd belt N merc on a hurt merc, arrival wait, leave with a dead merc (needs the user and a d2d run)
