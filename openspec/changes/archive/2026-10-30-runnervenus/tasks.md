# Tasks

```yaml
change: runnervenus
status: archived
agent: runnervenus
branch: agent/runnervenus
worktree: ~/code/d2lod-wt/runnervenus
owns:
  - docs/process/ci-runners-plan.md
exclusive:
  - runner:windows-worker
gates:
  - "unobtrusive: hidden, BelowNormal, capacity 1, no system settings changed"
merge: 8ea5036
result: "a Windows worker runner (Windows, per-user, S4U scheduled task ForgeRunner, hidden, BelowNormal, capacity 1, labels windows-worker/windows/windows-latest/windows-amd64); cross-compiled v13.2.0 binary (no official Windows asset); smoke ok; Rust/Inno Setup/MSVC/Tauri not installed"
```

- [x] inspect windows-worker read-only
- [x] portable Go and MinGit with sha256, runner exe
- [x] register user-level, task S4U hidden BelowNormal
- [x] smoke and cleanup
