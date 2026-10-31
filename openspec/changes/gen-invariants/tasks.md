# Tasks

```yaml
change: gen-invariants
status: in-progress
agent: geninv
branch: agent/geninv
worktree: ~/code/d2lod-wt/geninv
owns:
  - cmd/d2dsoak/**
  - internal/d2dsoakstat/**
  - docs/d2d-soak.md
  - docs/geninv.md
  - docs/probes/geninv/**
  - openspec/changes/gen-invariants/**
exclusive: []
gates:
  - "go test -p 1 ./cmd/d2dsoak ./internal/d2dsoakstat passes: fixture tests for clean, hung, wedged, stall-ladder and early-exit runs, a golden report and the holdout grouping"
  - "offline on the real logs: navsoak and navsoak2 report VIOLATED(WEDGE,PROGRESS), the 8 m2early runs and the navsoak3 prefix report their real counters (raw output in docs/probes/geninv)"
  - "whole-repo vet and short tests plus race on cmd/d2dsoak and internal/d2dsoakstat pass on the worker host; changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict"
merge:
result:
```

## 1. Work

- [x] 1.1 Read the generality study, soak-gates, fidelity-gate, death-rule, bot-loop specs and the overnight logs; pick the log signals for each invariant
- [x] 1.2 `ScanInvariants`, the seven invariants with thresholds and provenance, `InvReport` (groups, coverage, verdict), wiring into run, extract, compare (text and -json)
- [x] 1.3 Tests: table fixtures per invariant, golden report, holdout grouping, run.json round trip, exit codes unchanged
- [x] 1.4 Offline validation on navsoak, navsoak2, navsoak3 (prefix) and the 8 m2early runs: docs/probes/geninv
- [x] 1.5 Delta spec soak-gates, docs/geninv.md, docs/d2d-soak.md section; vet, short tests, race on the worker host, changecheck, leakcheck, validate
