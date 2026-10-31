# Tasks

```yaml
change: runnertower
status: archived
agent: runnertower
branch: agent/runnertower
worktree: ~/code/d2lod-wt/runnertower
owns:
  - tools/ci/runner/**
  - docs/process/ci-runners-plan.md
exclusive:
  - nas-runner
gates:
  - "container visible in the NAS Docker tab (managed label + template)"
  - "no docker.sock, no ports, no secrets in the repo"
merge: e768b29
result: "the NAS runner container forge-runner, labels linux/ubuntu-*; smoke ok"
```

## 1. Work

- [x] 1.1 Image, config, template
- [x] 1.2 Register (user scope) and start; online
- [x] 1.3 Smoke test on branch ci-smoke, delete branch
- [x] 1.4 Docs: Applied: NAS, rollback
