# Tasks

```yaml
change: d2dfarm
status: archived
agent: d2dfarm
branch: agent/d2dfarm
worktree: ~/code/d2lod-wt/d2dfarm
owns:
  - tools/d2dfarm*
  - docs/process/d2dfarm.md
  - openspec/changes/d2dfarm/**
exclusive:
  - d2d-lab-binary
gates:
  - "tools/d2dfarm-test.sh passes with a fake ssh"
  - "cross-host replay of one scripted scenario matches on both node types, or the first divergence is reported"
merge: 6aabf22
result: 'd2dfarm: node registry, probes, run/status/collect/verdict/stop by pgid, WSL node, cross-host determinism: lab6 sha identical on the worker host and a Windows worker, scripted replay 8/8 checkpoints match on a Windows worker; swappable transport (ssh now; forge Actions and mesh later); spec d2d-farm'
```

## 1. Work

- [x] 1.1 node registry, probe, pause/resume, nodes listing
- [x] 1.2 queue, dispatcher, collect, stop by pid, verdict over collected runs
- [x] 1.3 spec d2d-farm and process doc with transport section
- [x] 1.4 cross-host determinism check
