# Tasks

```yaml
change: verifyresistq
status: archived
agent: verifyresistq
branch: agent/verifyresistq
worktree: ~/code/d2lod-wt/verifyresistq
skip_specs: true
owns:
  - docs/research/verifyresistq.md
  - tools/emu/vq_*.py
  - openspec/changes/verifyresistq/**
exclusive: []
gates:
  - own harness run on two seeds before the author's rows are compared
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
merge: bbf5986
```

- [x] 1 Read FUN_005a1370 and FUN_0057cb00 in the disassembly
- [x] 2 Q3 harness vq_mods.py, seeds 11 and 2025, 0 mismatches
- [x] 3 Q9 harness vq_gethit.py, seeds 11 and 2025
- [x] 4 Q7 values from Skills.txt (vq_curse.py)
- [x] 5 Write docs/research/verifyresistq.md
- [ ] 6 changecheck and leakcheck (run before merge)
