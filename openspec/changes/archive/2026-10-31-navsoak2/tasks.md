# Tasks

```yaml
change: navsoak2
status: archived
agent: navsoak2
branch: agent/navsoak2
worktree: ~/code/d2lod-wt/navsoak2
owns:
  - docs/probes/navsoak2/**
  - openspec/changes/archive/2026-10-31-navsoak2/**
exclusive: []
gates:
  - "soak started on the worker host and healthy for the first 20 minutes"
merge: 5001721
result: soak started 2026-10-31 02:58:35 (worker host clock) in slot 1, verified live; see docs/probes/navsoak2/README.md
```

## 1. Work

- [x] 1.1 build d2bot from origin/main d82bd47 to a new binary and start the soak (seed 5, Navy2)
- [x] 1.2 verify the first 20 minutes
- [x] 1.3 write the handoff README
