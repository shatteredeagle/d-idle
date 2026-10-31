# Tasks

```yaml
change: livelockfix
status: archived
merge: b727fb4
result: 'offline only, not verified live: per-game cleared set reset in the task path (fresh-game livelock); Game.exe-affecting, live 2 h d2d run still pending'
agent: LIVELOCKFIX
branch: agent/livelockfix
worktree: ~/code/d2lod-wt/livelockfix
owns:
  - internal/bot/task.go
  - internal/bot/gate_plan_test.go
  - internal/bot/task_test.go
```

- [x] reproduce offline (test fails on master)
- [x] fix: syncGames in LevelTo and World
- [ ] live 2 h virtual run on d2d lab6 (not done at merge; Game.exe-affecting: needs live check)

result: offline only, not verified live.
