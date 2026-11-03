# Tasks

```yaml
change: verifydmg2
status: archived
agent: verifydmg2
branch: agent/verifydmg2
worktree: d2lod-wt/verifydmg2
skip_specs: true
owns:
  - openspec/changes/verifydmg2/**
  - docs/research/verify-dmgroll2.md
  - docs/research/verify-dmgroll2-run.txt
  - tools/emu/v2_*
  - tools/emu/vd_xref.py
exclusive: []
gates:
  - "own differential: r2 0 differences on roll, reduce and absorb (8 Act 1 classes, 10,000 frames each, own seed)"
  - "changecheck 0 warnings; leakcheck -mode fail 0"
result: "docs and tools only; see card result"
credit:
  component: Monster physical damage roll (second verification)
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 158ac61
```

## 1. Work

- [x] 1.1 Read the r2 commit (private clone in scratch), hygiene, merge-tree with PR 178's head
- [x] 1.2 game.exe reads: FUN_005ca6c0 (Energy Shield), FUN_0057c6c0 (life under one point), FUN_005c4d10 / FUN_005cce80 (fixed records)
- [x] 1.3 Own harness: game.exe roll natively plus game-order model, d2d base and r2 built from the trees' headers
- [x] 1.4 CI syntax parity (g++ / clang++) and the suites on the author's build of an identical tree
- [x] 1.5 docs/research/verify-dmgroll2.md, merge-lines.txt, OWNERS lines
