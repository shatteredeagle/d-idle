# Tasks

```yaml
change: generality
status: archived
agent: generality
branch: agent/generality
worktree: ~/code/d2lod-wt/generality
owns:
  - docs/research/bot-generality.md
exclusive: []
gates:
  - "docs/research/bot-generality.md <= 140 lines with evidence table, vocabulary, layer table, experiments, plan, SME notes"
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge: d348529
result: "docs only, not verified live (no game run): study written, SMEs consulted"
```

## 1. Work

- [x] 1.1 measure class, act, clvl, difficulty and engine fitting by grep counts with file:line (doc section 2)
- [x] 1.2 vocabulary, target architecture, experiment ranking, card plan (sections 3 to 6)
- [x] 1.3 consult d2-sme and spec-sme, record agreement (section 7)
- [x] 1.4 changecheck, leakcheck, openspec validate
