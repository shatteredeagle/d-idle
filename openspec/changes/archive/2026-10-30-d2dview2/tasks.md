# Tasks

```yaml
change: d2dview2
status: archived
agent: d2dview2
branch: agent/d2dview2
worktree: ~/code/d2lod-wt/d2dview2
owns:
  - tools/d2dview-demo.sh
  - cmd/d2watch/web/app.js
  - internal/d2denv/env.go
exclusive: []
gates:
  - "every claim tagged V, S or U"
merge: 3ba7943
result: 'd2watch d2d tile (not deployed), --d2d-dial flag, tools/d2dview-demo.sh, demo measured: fast mode 38-56x no page, 22-29x with page open, viewer ~18% of a core; watch mode 2.1x'
```

## 1. Work

- [x] 1.1 --d2d-dial flag with test (also touches cmd/d2bot/d2dbackend.go and its test: owned by d2d-default-backend, 3 lines + test)
- [x] 1.2 d2watch tile
- [x] 1.3 tools/d2dview-demo.sh
- [x] 1.4 live demo run, speed and CPU numbers
