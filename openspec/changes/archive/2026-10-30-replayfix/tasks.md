# Tasks

```yaml
change: replayfix
status: archived
merge: c682f2b
agent: replayfix
branch: agent/replayfix
worktree: ~/code/d2lod-wt/replayfix
owns:
  - internal/d2dreplay/**
  - cmd/d2dreplay/**
  - cmd/d2dview/mux.go
  - cmd/d2dview/view_test.go
  - openspec/changes/replayfix/**
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dreplay ./cmd/d2dreplay ./cmd/d2dview passes; -race on d2dreplay and d2dview (the worker host)"
  - "changecheck 0 warnings"
result: 'replay tap stops on signal/-max/-idle and always writes the trailer; reader validates and caps (256 MB, 2M entries, 1 MB lines, 1e6 steps); verb/arg allow-lists; launcher uses the slot pool; render -out hardened; d2dview watch speed clamped 0.5-1000 and pacing sleeps outside the lock; fuzz seeds. Offline only, not run live.'
```

## 1. Fixes (each with a failing test first)
- [x] 1.1 tap shutdown, trailer, busy, socket safety
- [x] 1.2 reader caps and validation, verb and args allow-lists, fuzz seeds
- [x] 1.3 launcher slot pool, release order, render -out
- [x] 1.4 d2dview speed clamp, pace outside lock, verb variants test
