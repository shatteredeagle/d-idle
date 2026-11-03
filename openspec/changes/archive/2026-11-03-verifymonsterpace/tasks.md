# Tasks

```yaml
change: verifymonsterpace
status: archived
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
result: "docs only, not live (no game.exe instance); READY WITH CHANGES: rebase onto current upstream main (CI clang-tidy fails on test_game_missiles.cpp at 6935cf64), PR-text fixes, split velocity vs paces recommended; see docs/research/pr-monsterpace-verifier.md"
merge: e1364ce
```

## 1. Work

- [x] 1.1 Diff, addresses against the disassembly
- [x] 1.2 Own oracles: velocity initialiser, paces, corpse reach
- [x] 1.3 Readers of the velocity stat and pace fields
- [x] 1.4 Det scenarios base vs head
- [x] 1.5 CI parity: warnings, MSVC hazards, clang-tidy step
- [x] 1.6 Suites, ctest, merges
- [x] 1.7 Draft claims vs counts; split; verdict
