# Tasks

```yaml
change: verifynmlevel
status: archived
agent: verifynmlevel
branch: agent/verifynmlevel
worktree: ~/code/d2lod-wt/verifynmlevel
skip_specs: true
owns:
  - docs/research/pr-nmlevel-verifier.md
  - docs/research/pr-nmlevel-verifier/**
  - openspec/changes/verifynmlevel/**
exclusive: []
gates:
  - own harness run on two seeds before the author's draft is compared
result: "docs only; several checks not run, listed in the report"
merge: 1d6dfd1
```

- [x] 1 Read FUN_00573cb0 / FUN_0061a1b0 / FUN_0061dca0 and the masks in the disassembly
- [x] 2 Own harness: game.exe populations (two seeds, every non-town level, three difficulties) against d2d base and head
- [x] 3 Caller and reader audit of make_monster / monster_stats
- [ ] 4 (not run) Det scenarios base and head (one hidden d2d slot)
- [ ] 5 (partly: tidy on three TUs, merge) CI parity: g++ / clang++ warnings, clang-tidy Names and includes, suites, merge against upstream main
- [x] 6 Compare with the author's draft; write docs/research/pr-nmlevel-verifier.md with the verdict
- [x] 7 changecheck and leakcheck
