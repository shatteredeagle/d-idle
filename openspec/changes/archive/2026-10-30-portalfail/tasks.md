# Tasks

```yaml
change: portalfail
status: archived
agent: portalfail
branch: agent/portalfail
worktree: ~/code/d2lod-wt/portalfail
owns:
  - internal/combat/fighter.go
  - internal/combat/catalog.go
  - internal/combat/portal_test.go
  - internal/combat/portalempty_test.go
  - internal/town/shop.go
  - internal/town/town_test.go
  - docs/earlygame.md
exclusive: []
gates:
  - "go test -race per package, whole repo on the worker host"
merge: b13b521
result: "root cause: an EMPTY Tome of Town Portal counted as a portal in combat, Safety kept choosing TownPortal: 159 failed attempts, 53 safety failures in 12 min on Fresha; fixed: empty tome is not a portal, 30 s silence after 3 failed casts, portal: attempt N failed because log, shop holds at most 2 spare scrolls unless rich"
```

## 1. Work

- [x] 1.1 Empty tome is not a portal; backoff after a failed cast; attempt log line
- [x] 1.2 Shop spare-scroll cap when poor
- [x] 1.3 Docs note
