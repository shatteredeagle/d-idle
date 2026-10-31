# Tasks

```yaml
change: runfix
status: archived
agent: runfix
branch: agent/runfix
worktree: ~/code/d2lod-wt/runfix
owns:
  - internal/d2dinput/run.go
  - internal/d2dinput/run_test.go
  - docs/d2d-run.md
exclusive: []
gates:
  - d2dsoak compare -gate v2 not worse than the walk-only arm on the same seeds
merge: 11e29e2
result: "run policy RunPolicy (opt-in D2D_RUN): stop <8 % stamina, resume 25 %; stamina model V live (max 74, run drain 3.9/s, regen 7.2/s standing; walk 1.85 vs run 2.8 cells/s); A/B pending (runab)"
```

- [x] stamina model probe on hidden d2d
- [x] run policy in d2dinput with fake tests
- [ ] before/after measurement, docs/d2d-run.md
