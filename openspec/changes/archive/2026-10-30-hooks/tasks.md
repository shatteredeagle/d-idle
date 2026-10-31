# Tasks

```yaml
change: hooks
status: archived
agent: hooks
branch: agent/hooks
worktree: ~/code/d2lod-wt/hooks
owns:
  - tools/hooks/**
  - tools/install-hooks.sh
  - tools/session-brief.sh
  - .claude/settings.json
  - openspec/.strict.yaml
  - docs/process/hooks.md
  - docs/process/ci-checks.md
  - cmd/changecheck/**
exclusive: []
gates:
  - "pre-push blocks a master push with a failing check"
  - "session brief < 100 ms"
merge: 457fa44
result: "pre-push hook, strict dates, session brief (offline tests only; not verified live)"
```

## 1. Work

- [x] 1.1 Strict dates, allowlist and tests in changecheck
- [x] 1.2 pre-push hook and install script
- [x] 1.3 Session brief and SessionStart wiring
- [x] 1.4 docs/process/hooks.md and ci-checks.md
