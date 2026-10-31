# Tasks

```yaml
change: m2-soakexit
status: proposed
agent: m2-soakexit
branch: agent/m2-soakexit
worktree: ~/code/d2lod-wt/m2-soakexit
owns:
  - cmd/d2dsoak/**
  - docs/d2d-soak.md
exclusive: []
gates:
  - "go test ./cmd/d2dsoak passes including TestVerdictExitCodes (PASS 0, INCONCLUSIVE 2, FAIL 3, NO-DATA 3) for run and compare"
  - "go test ./cmd/d2dsoak passes including TestExcludedSeedsAreNotExposure (a seed with 0 minute lines adds 0 h and appears under excluded seeds)"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [ ] 1.1 Read docs/research/code-review-2026-10-30.md section 9 (H1) and reproduce the exit-0 leak with a fake run (`d2dsoak run -fake`); verified by a failing test first
- [ ] 1.2 Exit codes per verdict for run and compare; TestVerdictExitCodes
- [ ] 1.3 Excluded seeds: 0 minute lines or an early d2d exit; TestExcludedSeedsAreNotExposure; report section
- [ ] 1.4 docs/d2d-soak.md exit-code table; changecheck, leakcheck, validate
