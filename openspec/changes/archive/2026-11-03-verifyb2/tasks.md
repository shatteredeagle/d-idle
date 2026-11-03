# Tasks

```yaml
change: verifyb2
status: archived
agent: pa-verifyb2
branch: agent/verifyb2
worktree: d2lod-wt/verifyb2
skip_specs: true
owns:
  - docs/research/verify-batch2.md
  - tools/emu/vb2_*
  - openspec/changes/verifyb2/**
exclusive: []
gates:
  - "own harness: game.exe natively vs the real d2d builds base and head, counted"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs and tools only; see card result"
credit:
  component: VERIFYB2
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
merge: f9aa452
```

## 1. Work

- [x] 1.1 Read FUN_0056c160, FUN_0056bfe0, FUN_005ca360, FUN_00554ea0, FUN_0064e7b0, FUN_0064dea0
- [x] 1.2 Own game-side harnesses (mana, warp spot search), own seeds
- [x] 1.3 Real d2d builds base vs heads, own probes, counts
- [x] 1.4 Commit hygiene, warnings, suites, merge with pull/178
- [x] 1.5 Compare with the author's counts; report docs/research/verify-batch2.md
