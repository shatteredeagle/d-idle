# Tasks

```yaml
change: m2-abdesign
status: in-progress
agent: m2-abdesign
branch: agent/m2-abdesign
worktree: ~/code/d2lod-wt/m2-abdesign
owns:
  - tools/ab/**
  - docs/ab-design.md
exclusive: []
gates:
  - "python3 -m unittest discover tools/ab passes (paired statistic on a fixture of 8 seeds x 2 arms; the required-runs formula)"
  - "docs/ab-design.md re-analyses the xpgap A/B (docs/probes/xpgap) with the paired statistic and states the runs per arm for a 10 percent effect"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result: 'tools/ab/paired.py (paired per-seed differences, t interval, runs_needed from the measured SD, verdict rule, exit codes), tools/ab/test_paired.py (12 tests, python3 -m unittest discover tools/ab), tools/ab/run.sh (interleaved arms via tools/xpgap/ab.sh, refuses under 8 seeds), docs/ab-design.md with the xpgap re-analysis (arms shared seeds 1-8). Finding: pairing helps only as far as the change leaves the run intact (arm correlation 0.03 to 0.95): 34 seeds per arm for a 10 percent effect when the change reshuffles the early route, 8 to 17 for town-only changes. cmd/d2dsoak and tools/d2dloop not changed (not owned). Offline only, no live run.'
```

## 1. Work

- [x] 1.1 Read docs/d2d-xp-gap.md (xpgap) and docs/probes/xpgap; confirm the arms used the same seeds
- [x] 1.2 tools/ab/paired.py and tests; the sample-size rule from the measured SD
- [x] 1.3 docs/ab-design.md with the worked example; changecheck, leakcheck, validate
