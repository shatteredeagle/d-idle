# Tasks

```yaml
change: streamview
status: archived
merge: 5217534
agent: streamview
branch: agent/streamview
worktree: ~/code/d2lod-wt/streamview
owns:
  - cmd/d2dview/**
  - docs/streamctl-d2d.md
  - tools/publish/**
  - cmd/d2watch/web/app.js
  - cmd/d2watch/web/style.css
exclusive: []
gates:
  - "go test ./cmd/d2dview ./cmd/d2watch passes"
  - "python3 -m unittest discover -s tests in tools/publish: no new failures"
result: "d2dview /stream (gameplay frames + map side by side, sanitized /api/pub feed), backend badges d2lod / d2d on all views, stream readiness doc: kiosk can capture http://127.0.0.1:8891/stream locally, NOT started; user must approve deploy + scene URL + streamctl live mode"
```

## 1. Work

- [x] 1.1 /stream page, /api/pub, tests
- [x] 1.2 badges in d2dview, d2watch (app.js, style.css), publish wording
- [x] 1.3 docs/streamctl-d2d.md readiness note
