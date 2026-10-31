# Tasks

```yaml
change: needsyou
status: archived
agent: needsyou
branch: agent/needsyou
worktree: ~/code/d2lod-wt/needsyou
owns:
  - docs/needs-you.md
  - tools/needs-you.sh
  - tools/agentstatus/changes.html
  - cmd/changecheck/**
exclusive: []
gates:
  - "changecheck 0 warnings, go test -race ./cmd/changecheck, sh -n tools/needs-you.sh"
merge: 5d2a999
result: "docs/needs-you.md + tools/needs-you.sh + board Waiting-on-you panel; not deployed until the board redeploy"
```

## 1. Work

- [x] 1.1 docs/needs-you.md and tools/needs-you.sh
- [x] 1.2 changecheck json needs_you
- [x] 1.3 Waiting on you panel on /changes
- [x] 1.4 dispatcher-brief line
