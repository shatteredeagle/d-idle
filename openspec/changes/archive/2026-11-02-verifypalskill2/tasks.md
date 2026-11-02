# Tasks

```yaml
change: verifypalskill2
status: archived
agent: verifypalskill2
branch: agent/verifypalskill2
worktree: d2lod-wt/verifypalskill2
skip_specs: true
owns:
  - docs/research/verifypalskill2.md
  - tools/emu/vp2*
  - openspec/changes/verifypalskill2/**
exclusive: []
gates:
  - own harness run on two seeds before the author's scripts are compared
result: "docs only; see the doc for tags, holds and limits"
merge: 5294558
```

- [x] 1 Read FUN_005ce9f0 and FUN_005ce8e0 in the disassembly
- [x] 2 Own harness (vp2.py, vp2_run.sh), seeds 5 and 23, 8,000 + 8,000 cases each
- [x] 3 Write docs/research/verifypalskill2.md with verdicts
- [x] 4 changecheck and leakcheck
