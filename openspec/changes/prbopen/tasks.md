# Tasks

```yaml
change: prbopen
status: in-progress
agent: prbopen
branch: agent/prbopen
worktree: ~/code/d2lod-wt/prbopen
skip_specs: true
owns:
  - docs/research/openpr-log.md
  - openspec/changes/prbopen/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits"
result: "PR 189 opened on psi29a/d2decomped; oracle rerun 560,000 frames 0 differ; log lines written; not a live-game job"
live: "not verified live (offline oracle on the worker host)"
```

- [x] rerun the oracle on the head
- [x] push the new branch and open one PR
- [x] log lines
