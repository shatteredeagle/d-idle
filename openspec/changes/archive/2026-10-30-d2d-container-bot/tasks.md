# Tasks

```yaml
change: d2d-container-bot
status: archived
agent: d2d-container-bot
branch: agent/d2d-container-bot
worktree: ~/code/d2lod-wt/d2d-container-bot
owns:
  - deploy/container/d2d/**
  - docs/d2d-container-bot.md
exclusive: []
gates:
  - "go test over the new packages passes"
  - "one slot runs in the container on the worker host for 1 h"
merge: archived-by-dispatch
result: "d2d container: built on the worker host (386 MB image), smoke PASS (ping, state, create character, save, restart, reload) with game data mounted :ro and no network; bot mode 90 s run ok; remaining tasks (1h gate, realtime mode, read-only root, fleet wizard, soak sidecar, optional Sentry patch) moved to docs/BACKLOG.md; open questions: GPL distribution (ADR 0010), ports, NAS template sketch (not installed)"
```

## 1. Work

- [ ] 1.1 d2bot entry for d2d with per-slot sockets (step 5)
- [ ] 1.2 d2fleet wizard for d2d (step 6)
- [ ] 1.3 viewer source via the screenshot verb (step 7)
- [ ] 1.4 soak driver sidecar (step 8)
