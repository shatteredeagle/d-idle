# Tasks

```yaml
change: detloop
status: archived
agent: DETLOOP
branch: agent/detloop
worktree: ~/code/d2lod-wt/detloop
owns:
  - docs/d2d-determinism.md
  - tools/d2ddet/**
  - cmd/d2ddet/**
  - internal/d2dctl/journal.go
  - internal/d2dctl/journal_test.go
  - internal/d2dctl/client.go
  - internal/d2denv/env.go
  - internal/bot/task.go
  - openspec/changes/detloop/**
exclusive:
  - d2d-lab-binary
gates:
  - "N=5 pairs of same-seed full-bot runs of 10000+ ticks: identical command streams and state hashes at every checkpoint, or the first divergence tick and its cause reported"
  - "changecheck 0 warnings"
merge: da4375d
result: "determinism: found the runner guard goroutine stepping the game on wall time (every same-seed pair diverged at tick 2): fixed (Pacer.Watch/LateClock.Watch); D2DCTL_JOURNAL journal + cmd/d2ddet compare; 2 pairs identical for ~2 virtual min then diverged at the fork link worker (poll_levels): one-line lockstep wait requested (in lab6 by forkrebase); Game.exe-affecting guard change: needs a G1/G2 check"
```

## 1. Work

- [x] 1.1 divergence finder: d2dctl journal (commands, ticks, reply digests, checkpoints), cmd/d2ddet compare, tools/d2ddet/pair.sh
- [x] 1.2 run same-seed pairs (5000 then 10000+ ticks), find and classify every first divergence
- [x] 1.3 fix the bot-side sources, keep all tests green, flag Game.exe-affecting edits
- [ ] 1.4 verify N=5 pairs (2 pairs ran: identical to tick 3002, diverge at the neighbour-link refetch; blocked on the fork request 4.1)
- [x] 1.5 docs/d2d-determinism.md: sources, fixes, remaining, fork requests, replay file and checkpoint format
