# Tasks

```yaml
change: detloop2
status: archived
agent: DETLOOP2
branch: agent/detloop2
worktree: ~/code/d2lod-wt/detloop2
owns:
  - docs/d2d-determinism.md
  - tools/d2ddet/**
  - cmd/d2ddet/**
  - internal/d2dreplay/**
  - internal/d2dctl/journal.go
  - internal/d2denv/env.go
  - internal/d2dcombat/rig.go
exclusive:
  - d2d-lab-binary
gates:
  - "5 of 5 same-seed pairs identical for 10000+ ticks"
merge: 956a15d
result: "d2d determinism: cause = d2dworld refetch limiter on wall time (d2denv never set Options.Now): 2-line fix; live: 3 same-seed runs record-identical to tick 12002 (was 2394); remaining diff only at the end-of-run stop (not tick-based yet); 5 of 5 / 50000 ticks not measured; full-bot replay on lab6 not run"
```

## 1. Work

- [x] find the cause of the differing `debug grid 5`
- [x] refetch timer on the lockstep clock
- [ ] pair run, mask start-up state fields
- [ ] full-bot replay past the first restart marker
