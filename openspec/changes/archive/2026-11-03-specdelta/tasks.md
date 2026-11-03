# Tasks

```yaml
change: specdelta
status: archived
agent: specdelta
branch: agent/specdelta
worktree: ~/code/d2lod-wt/specdelta
owns:
  - openspec/changes/specdelta/**
  - docs/research/specdelta-rejudge.md
  - internal/d2dsoakstat/invariants.go
  - internal/d2dsoakstat/invariants_test.go
  - internal/d2dsoakstat/invreport.go
  - internal/d2dsoakstat/testdata/invreport.golden
  - docs/d2d-soak.md
  - docs/geninv.md
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dsoakstat ./cmd/d2dsoak passes"
  - "python3 -I tools/tests/gate_judge_xp_test.py in the ops repo passes"
  - "changecheck -cli 0 warnings; leakcheck 0 hits; tools/ci/check.sh ok"
merge: 4c99d0b
result: "go tests d2dsoakstat and cmd/d2dsoak, ops gate_judge_xp_test; openspec strict validate 56/56"
credit:
  author: owner
  agents: [Claude Code]
```

## 1. Work

- [x] 1.1 delta spec for both requirements (tools/openspec.sh validate --all --strict)
- [x] 1.2 d2dsoakstat BELT uses funded windows (BeltFundedMin); tests: poverty, funded, mixed, old archive, golden
- [x] 1.3 ops tools/gate-judge.py xp_gate1 plus unit test (ops master 979b83d)
- [x] 1.4 docs: decisions.md two lines, d2d-soak.md, geninv.md
- [x] 1.5 re-judge table docs/research/specdelta-rejudge.md (not verified live; replayed from the archives)
