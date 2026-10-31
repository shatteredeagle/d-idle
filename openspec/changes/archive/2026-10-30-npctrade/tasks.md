# Tasks

```yaml
change: npctrade
status: archived
agent: npctrade
branch: agent/npctrade
worktree: ~/code/d2lod-wt/npctrade
owns:
  - internal/town/npc*.go
  - internal/town/sim/**
  - docs/npc.md
exclusive: []
gates:
  - "root cause of inert dialog clicks is U until a live run with the new log line"
merge: bdf5fb4
result: "trade-stage failures (18: G2 15, Fresha 3): dialog ignores clicks, root cause unproven (U); fix bounds the damage: 1.5 s proof wait, ignored-click detection, Esc after 3, visit capped ~9 s; Gheed first adjust +5; live verification pending"
```

## 1. Work

- [x] 1.1 Timelines and counts from the G2 and Fresha logs
- [x] 1.2 Fail-fast pickEntry, Gheed bias, sim knob MenuInert, tests
- [ ] 1.3 Live check of the new log line (not done, no game)
