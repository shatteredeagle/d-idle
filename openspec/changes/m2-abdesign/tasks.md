# Tasks

```yaml
change: m2-abdesign
status: proposed
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
result:
```

## 1. Work

- [ ] 1.1 Read docs/d2d-xp-gap.md (xpgap) and docs/probes/xpgap; confirm the arms used the same seeds
- [ ] 1.2 tools/ab/paired.py and tests; the sample-size rule from the measured SD
- [ ] 1.3 docs/ab-design.md with the worked example; changecheck, leakcheck, validate
