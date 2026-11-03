# Tasks

```yaml
change: b2winworker
status: archived
agent: b2winworker
branch: agent/b2winworker
worktree: d2lod-wt/b2winworker
skip_specs: true
owns:
  - docs/research/b2-winworker-rerun.md
  - openspec/changes/b2winworker/**
exclusive: []
gates:
  - "MSVC 14.51 and WSL results for local/batch2-bundle-r3, local/batch2-mana-teleport, local/batch2-monsterdmg-pr recorded with the exact heads"
  - "changecheck 0 warnings, leakcheck 0 hits"
result: "PARTIAL: mana/Teleport seed 9 and v4 not run; docs only"
credit:
  component: Batch 2 re-run on the Windows build worker
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 54ee48a
```

## 1. Work

- [x] 1.1 MSVC 14.51 lane for base 005c7cab and the three branches (build, warnings, ctest)
- [x] 1.2 WSL Linux lane (tools/parity.sh stages) for the three branches
- [ ] 1.3 Counted differentials in WSL: damage-roll sets done (mdmg2, v3); mana, Teleport seed 9, v4 NOT run
- [x] 1.4 docs/research/b2-winworker-rerun.md
