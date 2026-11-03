# Tasks

```yaml
change: verifydmg3
status: archived
agent: verifydmg3
branch: agent/verifydmg3
worktree: d2lod-wt/verifydmg3
skip_specs: true
owns:
  - openspec/changes/verifydmg3/**
  - docs/research/verify-dmgroll3.md
  - docs/research/verify-dmgroll3-run.txt
  - tools/emu/w3_*
exclusive: []
gates:
  - "own differential: r3 against game.exe's roll and the verifier's game-order model, own seed, every set; never worse than base"
  - "changecheck 0 warnings; leakcheck -mode fail 0"
result: "docs/tools only"
credit:
  component: Monster physical damage roll (third verification)
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 42e9e2f
```

## 1. Work

- [x] 1.1 Read the r3 commit (private clone in scratch), hygiene, merge-tree with PR 178's head
- [x] 1.2 game.exe reads: FUN_0057c6c0 and its two fixed-record callers, FUN_0057c1e0, the event list (FUN_005c0ad0, FUN_005c0c30), Energy Shield FUN_005ca6c0
- [x] 1.3 Own differential: base, r2, r3 and r3 without Fight::absorb's guard, every set
- [x] 1.4 CI syntax parity (g++ / clang++) and the suites on the author's build of an identical tree
- [x] 1.5 docs/research/verify-dmgroll3.md, merge-lines.txt, OWNERS lines
