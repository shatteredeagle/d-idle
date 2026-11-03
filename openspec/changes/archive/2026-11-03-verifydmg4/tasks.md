# Tasks

```yaml
change: verifydmg4
status: archived
agent: verifydmg4
branch: agent/verifydmg4
worktree: d2lod-wt/verifydmg4
skip_specs: true
owns:
  - openspec/changes/verifydmg4/**
  - docs/research/verify-dmgroll4.md
  - docs/research/verify-dmgroll4-run.txt
  - tools/emu/v4_*
exclusive: []
gates:
  - "own differential through the real Fight::absorb: r4 against game.exe's roll and the game-order model, own seed, every set; never worse than base"
  - "changecheck 0 warnings; leakcheck -mode fail 0"
result: "docs and tools only; see card result"
credit:
  component: Monster physical damage roll (fourth verification)
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 9c7bf26
```

## 1. Work

- [x] 1.1 Read the r4 commit (private clone in scratch), hygiene, merge-tree with PR 178's head
- [x] 1.2 Probe linked against the r4 libgame.a: the real Fight::absorb and Fight::hurt_side per case
- [x] 1.3 Own differential (seed 161803, 354,000 frames): base, r4 real, r4 glue; both absorb orders
- [x] 1.4 CI syntax parity (g++ / clang++) and the suites on the author's build of an identical tree
- [x] 1.5 docs/research/verify-dmgroll4.md, merge-lines.txt, OWNERS lines
