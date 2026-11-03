# Tasks

```yaml
change: soak-minutes-virtual
status: archived
skip_specs: false
agent: specmin
branch: agent/specmin
worktree: ~/code/d2lod-wt/specmin
owns:
  - openspec/changes/soak-minutes-virtual/**
  - internal/d2dsoakstat/invariants.go
  - internal/d2dsoakstat/invariants_test.go
  - internal/d2dsoakstat/invreport.go
  - internal/d2dsoakstat/minute.go
  - internal/d2dsoakstat/testdata/invreport.golden
  - docs/probes/postfix-soak2.md
  - docs/review-strategy.md
  - docs/d2d-soak.md
  - docs/geninv.md
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dsoakstat ./cmd/d2dsoak passes, incl. TestInvariantsGate1PostfixSoak2 (27 and 26 fail) and the clean synthetic log (passes)"
  - "tools/openspec.sh validate --all --strict passes; changecheck -cli 0 warnings; leakcheck -mode fail 0 hits"
merge: 12c11b9
result: "offline; no live run; spec delta applied by housekeep at archive (task 1.4 live proof still open)"
```

## 1. Work

- [x] 1.1 delta spec soak-gates (virtual minutes, belt 5, no kill and no XP 10), scenario per criterion
- [x] 1.2 d2dsoak: BELT and NOKILL counters and codes, printed with limits; unit tests with the soak 2 log and synthetic fixtures
- [x] 1.3 docs: postfix-soak2 decision section, review-strategy status, d2d-soak and geninv notes, decisions.md line, INDEX, CHANGELOG, BACKLOG
- [ ] 1.4 live proof: the next Gate 1 soak report shows the gate 1 line with both stretches (coordinator)
- [ ] 1.5 apply the spec delta at archive (coordinator)
