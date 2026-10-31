# Tasks

```yaml
change: m2-soakexit
status: in-progress
agent: m2-soakexit
branch: agent/m2-soakexit
worktree: ~/code/d2lod-wt/m2-soakexit
owns:
  - cmd/d2dsoak/**
  - docs/d2d-soak.md
  - internal/d2dsoakstat/**
exclusive: []
gates:
  - "go test ./cmd/d2dsoak passes including TestVerdictExitCodes (PASS 0, INCONCLUSIVE 2, FAIL 3, NO-DATA 3) for run and compare"
  - "go test ./cmd/d2dsoak passes including TestExcludedSeedsAreNotExposure (a seed with 0 minute lines adds 0 h and appears under excluded seeds)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result: 'Much of this had merged earlier (soakfix: exit codes 0/2/3 for run and compare incl. -json, early/empty seeds invalid and excluded from hours, duplicate seeds rejected, min-sample INCONCLUSIVE, NaN/negative guards). Delta here: NO-DATA verdict and exit 3 (no usable seed; unknown verdicts fail closed), run with all seeds dead is NO-DATA not an I/O error, excluded seeds section + excludedSeeds/seeds in JSON, seed count excludes them, last line names the code. Tests TestVerdictExitCodes, TestExcludedSeedsAreNotExposure, TestNoUsableSeedIsNoData. Offline only, not verified live (d2d-only tool, no game needed).'
```

## 1. Work

- [x] 1.1 Read docs/research/code-review-2026-10-30.md section 9 (H1) and reproduce the exit-0 leak with a fake run (`d2dsoak run -fake`); verified by a failing test first
- [x] 1.2 Exit codes per verdict for run and compare; TestVerdictExitCodes
- [x] 1.3 Excluded seeds: 0 minute lines or an early d2d exit; TestExcludedSeedsAreNotExposure; report section
- [x] 1.4 docs/d2d-soak.md exit-code table; changecheck, leakcheck, validate
