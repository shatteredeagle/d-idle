# Tasks

```yaml
change: m2-roam
status: deferred
agent: m2-roam
branch: agent/m2-roam
worktree: ~/code/d2lod-wt/m2-roam
owns:
  - internal/nav/roam.go
  - internal/nav/roam_test.go
  - internal/nav/exit.go
  - internal/nav/exit_test.go
  - internal/bot/goal.go
  - internal/bot/emptyarea_test.go
  - docs/roam.md
exclusive:
  - soak-slot:3
gates:
  - "before/after on 4 seeds x 2 virtual hours per arm: walk share of kill time at most 55 percent (was 77) and `clear killed nothing` at most 0.15 per kill (was 0.30), same binary, interleaved arms, raw in docs/probes/roam"
  - "go test -p 1 ./internal/nav ./internal/bot passes; go test ./internal/env -run EveryKnob passes with the new default"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes; tools/agentreap.sh m2-roam leaves nothing"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/d2d-soak-full-fix.md kill-time section, docs/research/per-kill-time-split.md tables 2 and 3, docs/killfix.md; extract the border-crossing and empty-sweep counts of 4 soakfull seeds as the before arm
- [ ] 1.2 Roam and exit changes with offline tests on recorded maps (internal/nav fixtures)
- [ ] 1.3 emptyAreaSec default on; EveryKnob test; docs/knobs.md regenerated
- [ ] 1.4 After arm on the same seeds; docs/roam.md with both tables; changecheck, leakcheck, validate; reap
