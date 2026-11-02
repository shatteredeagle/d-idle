# Tasks

```yaml
change: verifyaidel
status: archived
agent: verifyaidel
branch: agent/verifyaidel
worktree: ~/code/d2lod-wt/verifyaidel
skip_specs: true
owns:
  - docs/research/verify-aidel.md
  - tools/emu/va_aidel.py
  - tools/emu/va_aidel_run.py
  - openspec/changes/verifyaidel/**
exclusive: []
gates:
  - changecheck and leakcheck clean
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
merge: 7f945d3
```

- [x] 1 Table per difficulty
- [x] 2 Own harness for FUN_005a73e0
- [x] 3 Trace doc
- [ ] 4 FUN_005a7c20 / freeze-end path (open)
