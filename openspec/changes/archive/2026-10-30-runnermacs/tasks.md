# Tasks

```yaml
change: runnermacs
status: archived
agent: runnermacs
branch: agent/runnermacs
worktree: ~/code/d2lod-wt/runnermacs
owns:
  - docs/process/ci-runners-plan.md
exclusive:
  - runner:mac-a
  - runner:mac-b
gates:
  - "registration token never printed or stored"
merge: f2f63e8
result: "Mac worker A labels+Go ok; Mac worker B registered but offline until the macOS Local Network Allow click"
```

## 1. Work

- [x] 1.1 Mac worker A: Go, labels, priority, restart
- [x] 1.2 Mac worker B: binary, Go, register, LaunchAgent
- [ ] 1.3 Mac worker B online (needs the user's Allow click)
- [x] 1.4 Smoke on label macos (Mac worker A), branch deleted
