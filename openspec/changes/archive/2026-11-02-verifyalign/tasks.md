# Tasks

```yaml
change: verifyalign
status: archived
agent: verifyalign
branch: agent/verifyalign
worktree: d2lod-wt/verifyalign
skip_specs: true
owns:
  - openspec/changes/verifyalign/**
  - docs/research/verify-alignment2.md
  - tools/emu/vfal_run.py
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs only; see the doc for tags and limits"
credit:
  component: verifyalign
  author: owner
  verified_by: []
  agents: [Claude Code]
merge: 1046ffd
```

## 1. Work

- [x] 1.1 Own disassembly reading of the gate, getter, setter, spawn map
- [x] 1.2 Own emulator harness, two seeds
- [x] 1.3 Verdict doc, card, merge lines, OWNERS
