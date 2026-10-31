# Tasks

```yaml
change: nowall
status: archived
agent: nowall
branch: agent/nowall
worktree: ~/code/d2lod-wt/nowall
owns:
  - internal/bot/stall.go
  - internal/bot/stall_test.go
  - internal/bot/observe.go
  - internal/bot/runner.go
  - internal/bot/loop.go
  - cmd/d2bot/live.go
  - cmd/d2bot/supervise.go
  - cmd/d2bot/stall_test.go
  - internal/env/tuning_bot.go
  - docs/nowall.md
exclusive: []
gates:
  - "go test -p 1 -run 'Stall|HaltErrorCampaignStalled' ./internal/bot ./cmd/d2bot passes; go test ./internal/env -run EveryKnob passes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: 950a153
result: "campaign stall ladder on the virtual clock (warn + fresh game after 30 min, exit status 5 after 30 more; knobs bot.stallWarnMin, bot.stallExitMin, 0 = off); offline tests with a fake clock, not verified live"
```

## 1. Work

- [x] 1.1 Measure the longest legitimate no-progress stretch in the navsoak logs (19 virtual min)
- [x] 1.2 Progress tracking, ladder, exit status 5, knobs
- [x] 1.3 Offline tests with a fake clock
- [x] 1.4 Spec delta; knobs.md regenerated; checks
