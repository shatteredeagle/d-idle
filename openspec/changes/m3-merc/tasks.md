# Tasks

```yaml
change: m3-merc
status: deferred
agent: m3-merc
branch: agent/m3-merc
worktree: ~/code/d2lod-wt/m3-merc
owns:
  - internal/merc/**
  - internal/bot/resource.go
  - internal/bot/merc_test.go
  - docs/merc-d2d.md
  - docs/probes/merc/**
exclusive:
  - soak-slot:3
gates:
  - "2 seeds x 4 virtual hours from clvl 9 after Blood Raven on hidden d2d: the Rogue claimed exactly once per seed, 0 paid Act 1 hires, merc alive at least 70 percent of field time, every revive logged with price and gold delta within the revive rule (raw in docs/probes/merc)"
  - "go test -p 1 ./internal/merc ./internal/bot -run 'Merc|Claim' passes; go test ./internal/env -run EveryKnob passes with the d2d default"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m3-merc leaves nothing"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/merc-wire.md, docs/merc.md, merc-policy spec and the grill 0003 outcome; failing tests first (claim once, revive rule, stop rule)
- [ ] 1.2 Revive, stop rule and merc potions wired for d2d; knob default per backend
- [ ] 1.3 2 x 4 virtual hour runs; docs/merc-d2d.md; changecheck, leakcheck, validate; reap
