# Tasks

```yaml
change: potionbug
status: archived
agent: potionbug
branch: agent/potionbug
worktree: ~/code/d2lod-wt/potionbug
owns:
  - internal/combat/fighter.go
  - internal/combat/portal_test.go
  - internal/combat/crowd_test.go
  - internal/bot/autopsy.go
  - internal/bot/autopsy_test.go
  - internal/bot/goldsweep.go
  - internal/bot/goldsweep_test.go
  - docs/potion-bug.md
exclusive: []
gates:
  - "go test per package passes"
merge: bba8a15
result: "Fresha deaths: town portal cast starvation (up to 3 tries of ~2 s with no drink; decision prefers the portal) and a gold sweep back into the pack that killed it; fixed: drink between portal tries, skip the sweep after a crowd, new skip-reason log; BUG tag heuristic coarse (suggest lowLifeSecs >= 2); not verified live"
```

- [x] Timeline of both deaths from the soak log
- [x] Fix drink starvation during a town portal cast, skipped-drink log
- [x] Gold sweep refuses a camped death spot
- [x] Verified live (result: not verified live, offline tests only)
