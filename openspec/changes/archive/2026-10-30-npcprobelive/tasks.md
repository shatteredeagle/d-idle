# Tasks

```yaml
change: npcprobelive
status: archived
agent: npcprobelive
branch: agent/npcprobelive
worktree: ~/code/d2lod-wt/npcprobelive
owns:
  - internal/env/tuning_town.go
  - docs/npc-skip.md
exclusive:
  - instance:i5
gates:
  - "each strategy has at least 3 first-contact samples; default set from the data"
merge: f80c70b
result: "no strategy skipped the speech; default 0 wait; samples small; first contact is once per NPC per character"
```

## 1. Work

- [x] 1.1 Three fresh characters, 12 first-contact visits, analysis, default set
