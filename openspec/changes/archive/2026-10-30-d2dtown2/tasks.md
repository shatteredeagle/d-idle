# Tasks

```yaml
change: d2dtown2
status: archived
agent: d2dtown2
branch: agent/d2dtown2
worktree: ~/code/d2lod-wt/d2dtown2
owns:
  - internal/d2dtown/**
  - internal/d2dctl/town_verbs.go
  - cmd/d2dshop/**
  - docs/d2d-town.md
exclusive:
  - d2d-lab-lock
gates:
  - "go test -race of d2dctl and d2dtown passes"
  - "each gap has live evidence or a written fork verb request"
merge: 15acc41
result: "NPC reach via ApproachNPC side-step, stash class fix, gold deposit/withdraw verified, repair button verified (original failure not reproduced), portal cast verified and enter by left click on the sprite; waypoint row choice, Tome of TP, shift-click open; fork verb requests in docs/d2d-town.md; new code not re-run live"
```

## 1. Work

- [x] 1.1 Repair, stash, NPC walk, portal, waypoint, tome probes
- [x] 1.2 Fixes with d2dctl fake tests
- [x] 1.3
- [ ] 1.4 Live re-run of d2dshop with the new verbs (lab was busy) docs/d2d-town.md and fork verb requests
