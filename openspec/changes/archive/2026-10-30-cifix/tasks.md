# Tasks

```yaml
change: cifix
status: archived
agent: cifix
branch: agent/cifix
worktree: ~/code/d2lod-wt/cifix
owns:
  - .forgejo/workflows/**
  - docs/ci.md
  - tools/winereap_test.sh
exclusive: []
gates:
  - "no runner is registered or changed, nothing pushed"
merge: 4f4e98d
result: "winereap_test lock stall fixed; CI on master/PR only; checks+test jobs"
```

## 1. Work

- [x] 1.1 Reproduce the Scripts failure on the worker host as the runner would
- [x] 1.2 Fix it, rework triggers and jobs
- [x] 1.3 Update docs/ci.md, verify locally
