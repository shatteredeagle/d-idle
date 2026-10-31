# Tasks

```yaml
change: d2d-container-build
status: archived
agent: d2d-container-build
branch: agent/d2d-container-build
worktree: ~/code/d2lod-wt/d2d-container-build
owns:
  - docs/d2d-container/**
  - deploy/d2d-container/**
exclusive: []
gates:
  - "docker build of the d2d stage succeeds on the worker host"
  - "container smoke test: ping, state, create character, save"
merge: archived-by-dispatch
result: "d2d container: built on the worker host (386 MB image), smoke PASS (ping, state, create character, save, restart, reload) with game data mounted :ro and no network; bot mode 90 s run ok; remaining tasks (1h gate, realtime mode, read-only root, fleet wizard, soak sidecar, optional Sentry patch) moved to docs/BACKLOG.md; open questions: GPL distribution (ADR 0010), ports, NAS template sketch (not installed)"
```

## 1. Work

- [x] 1.1 vendor bundle script and context assembler (risks step 1)
- [x] 1.2 first d2d stage build on the worker host, g++ 15 or 16 (step 2)
- [ ] 1.3 optional fork patch: Sentry optional, launcher skipped (step 3; fork work pushed to <owner>/d2d-fork the same day)
- [x] 1.4 runtime image: notices, SOURCE.txt, non-root, tini, healthcheck; smoke with MPQs (step 4)
