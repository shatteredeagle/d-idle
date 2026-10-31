# Tasks

```yaml
change: killgap
status: archived
agent: killgap
branch: agent/killgap
worktree: ~/code/d2lod-wt/killgap
owns:
  - internal/bot/killtime.go
  - internal/bot/killtime_test.go
  - internal/bot/observe.go
  - internal/d2dsoakstat/killtime.go
  - internal/d2dsoakstat/killtime_test.go
  - cmd/d2dsoak/killtime.go
  - docs/killgap.md
exclusive:
  - d2d-lab-binary
gates:
  - "before/after seconds per kill on the same 4 seeds"
merge: 3b25667
result: "per-kill timing instrument (killtime line; d2dsoak extract buckets); baseline 3640 XP/h, 156 kills/h, 23 s per kill (walk 17, wait 5.3, cast 0.8); causes: empty-area roaming, cave-mouth shuttle, nav stuck; first fix attempt reverted"
```

## 1. Work

- [x] 1.1 killtime instrument and extract
- [x] 1.2 measure on d2d (4 seeds)
- [x] 1.3 one fix tried (nav explore frontier exit clearance), re-measured on 3 seeds, no gain, reverted
