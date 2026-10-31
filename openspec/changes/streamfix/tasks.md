# Tasks

```yaml
change: streamfix
status: in-progress
agent: streamfix
branch: agent/streamfix
worktree: ~/code/d2lod-wt/streamfix
owns:
  - openspec/changes/streamfix/**
  - cmd/d2dview/main.go
  - cmd/d2dview/pub.go
  - cmd/d2dview/view_test.go
  - cmd/d2dview/web/stream.html
  - tools/d2dview-demo.sh
exclusive: []
gates:
  - "go test -p 1 ./cmd/d2dview passes"
  - "tools/openspec.sh validate --all --strict passes"
merge: ""
result: "Built and verified live on the stream kiosk (:120): badge, engine, jpeg feed 0.4 MB/s, pane non-black for 10 min after swap. Not unit-tested in a browser."
```

- [x] 1 badge flag, engine field, reconnect logic, tests
- [x] 2 demo --badge
- [x] 3 live swap and verify (not verified live yet)
