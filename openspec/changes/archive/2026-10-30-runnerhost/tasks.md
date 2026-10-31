# Tasks

```yaml
change: runnerhost
status: archived
merge: f30d089
agent: runnerhost
branch: agent/runnerhost
worktree: ~/code/d2lod-wt/runnerhost
owns:
  - docs/process/ci-runners-worker.md
exclusive:
  - runner:worker
gates:
  - "never kill a running job: restart only when the runner is idle"
result: "the worker host keeps only label worker; routing test ok"
```

- [x] backup config.yml and .runner
- [x] wait for idle, set labels, restart user unit
- [x] routing test on throwaway branch ci-route (branch deleted)
- [x] doc
