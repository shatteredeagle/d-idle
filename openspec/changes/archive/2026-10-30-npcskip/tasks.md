# Tasks

```yaml
change: npcskip
status: archived
agent: npcskip
branch: agent/npcskip
worktree: ~/code/d2lod-wt/npcskip
owns:
  - internal/town/**
  - cmd/npcprobe/**
exclusive: []
gates:
  - "live probe on a fresh character picks the default strategy (pending)"
merge: 0980265
result: "merged offline-verified; default npc; the live probe decides the final default"
```

## 1. Work

- [x] 1.1 Strategies, logging, special-case helpers, probe tool, offline tests
- [x] 1.2 Live probe on three fresh characters (docs/npc-skip.md LIVE RESULT): no strategy skips the speech; default set to 0 (wait), the only one with no failures
