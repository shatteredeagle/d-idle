# Tasks

```yaml
change: d2dview
status: archived
agent: d2dview
branch: agent/d2dview
worktree: ~/code/d2lod-wt/d2dview
owns:
  - cmd/d2dview/**
  - docs/d2d-view.md
exclusive: []
gates:
  - "the viewer never sends a write verb (readOnly filter, offline test)"
  - "every claim tagged V, S or U"
merge: 3ba7943
result: 'cmd/d2dview tap (only connection to d2d, bot socket unfiltered, viewer socket read-only whitelist + screenshot); map page + sampled MJPEG; watch mode ~2x'
```

## 1. Work

- [x] 1.1 tap mux with read-only viewer socket
- [x] 1.2 map, state and frame feeds, web page, fast and watch modes
- [x] 1.3 offline tests (fake d2d)
- [x] 1.4 live check against a running d2d (see docs/d2d-view.md, Live)
- [ ] 1.5 d2watch tile (needs an owner for cmd/d2watch, request in docs/d2d-view.md)
