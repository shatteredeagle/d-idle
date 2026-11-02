# Tasks

```yaml
change: verifyskmelee2
status: archived
agent: verifyskmelee2
branch: agent/verifyskmelee2
worktree: ~/code/d2lod-wt/verifyskmelee2
skip_specs: true
owns:
  - docs/research/verify-skmelee2.md
  - tools/emu/vm2_*
  - openspec/changes/verifyskmelee2/**
exclusive: []
gates:
  - own harness run on two seeds before the author's trace is compared
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
merge: c40e7e6
```

- [x] 1 Read FUN_0057b420 / FUN_0057d9b0 / FUN_0057b660 in the disassembly
- [x] 2 Own harness (vm2_*), seeds 11 and 3, 20,000 cases each
- [x] 3 Write docs/research/verify-skmelee2.md with verdicts and differences from the author
- [x] 4 changecheck and leakcheck
