# Tasks

```yaml
change: verifymonsterpace
status: in-progress
agent: verifymonsterpace
branch: agent/verifymonsterpace
worktree: ~/code/d2lod-wt/verifymonsterpace
skip_specs: true
owns:
  - openspec/changes/verifymonsterpace/**
  - docs/research/pr-monsterpace-verifier.md
  - docs/research/pr-monsterpace-verifier/**
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "in progress"
```

## 1. Work

- [ ] 1.1 Diff, addresses against the disassembly
- [ ] 1.2 Own oracles: velocity initialiser, paces, corpse reach
- [ ] 1.3 Readers of the velocity stat and pace fields
- [ ] 1.4 Det scenarios base vs head
- [ ] 1.5 CI parity: warnings, MSVC hazards, clang-tidy step
- [ ] 1.6 Suites, ctest, merges
- [ ] 1.7 Draft claims vs counts; split; verdict
