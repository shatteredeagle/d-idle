# Tasks

```yaml
change: m3-readiness-model
status: deferred
agent: m3-readiness-model
branch: agent/m3-readiness-model
worktree: ~/code/d2lod-wt/m3-readiness-model
owns:
  - internal/readiness/**
  - cmd/d2ready/**
  - docs/readiness.md
exclusive: []
gates:
  - "offline replay of docs/plan/death-data.md and the soak death logs rates all 9 Monastery road deaths (clvl 11-12, 2-potion kit) as not ready and agrees with the m3-andariel fight results"
  - "go test -p 1 ./internal/readiness passes; go test ./internal/env -run EveryKnob passes with the new knobs"
  - "the guard consults the model only when its enable knob is 1 and never goes more than 2 clvl below the merc-policy floors (test)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Offline calibration harness: replay the death data and logs, print power, danger and verdict per death and per m3-andariel fight; failing tests first (the 9 road deaths are not ready)
- [ ] 1.2 The model and its knobs (weights, margin, enable, max clvl below the floor); scenarios from the road deaths; docs/readiness.md with V/U tags
- [ ] 1.3 Route-guard integration behind the enable knob, floors kept as the safety net; changecheck, leakcheck, validate
- [ ] 1.4 Open questions: model weights and margin; Andariel max-life floor; record in docs/readiness.md what stayed U
