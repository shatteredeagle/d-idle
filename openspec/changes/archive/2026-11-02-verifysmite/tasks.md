# Tasks

```yaml
change: verifysmite
status: archived
agent: verifysmite
branch: agent/verifysmite
worktree: d2lod-wt/verifysmite
skip_specs: true
owns:
  - docs/research/verify-smiteblock.md
  - docs/research/verify-smiteblock/**
  - openspec/changes/verifysmite/**
exclusive: []
gates:
  - own game.exe run before the author's trace is read
result: "docs and records"
merge: 894788d
```

- [x] 1 Own disassembly and do-func table
- [x] 2 Own end-to-end oracle of FUN_005ce9f0 (forced vs unforced) and the damage formula
- [x] 3 Other FUN_0057ec10 callers (Zeal, Sacrifice, Charge, Blessed Hammer)
- [ ] 4 d2d build and probe on upstream main (NOT RUN: time; read only)
- [x] 5 Proposed diff, tests, CI parity, bundle interaction
- [x] 6 Verdict; changecheck and leakcheck
