# Tasks

```yaml
change: d2d-default-backend
status: archived
agent: d2d-default-backend
branch: agent/d2d-default-backend
worktree: ~/code/d2lod-wt/d2d-default-backend
owns:
  - cmd/d2bot/d2dbackend_default_test.go
# owns released to fix-restart for the SavedExp wiring in d2dbackend.go; re-claim it before editing the file again
exclusive: []
gates:
  - "go test ./cmd/d2bot -run Backend passes"
  - "bot started with no flag uses d2d"
merge: none (dropped)
result: "DROPPED 2026-11-04 by user decision: the backend flip never happened (cmd/d2bot/d2dbackend.go still defaults --backend to game); ADR 0015 makes d2d the lab and game.exe the acceptance target; nothing implemented"
```

## 1. Work

- [ ] 1.1 flip the default (env D2BOT_BACKEND keeps working), game stays selectable
- [ ] 1.2 go test ./cmd/d2bot passes with both defaults covered
- [ ] 1.3 apply the spec delta when this merges
