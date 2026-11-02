# Tasks

```yaml
change: verifyskmana
status: archived
agent: verifyskmana
branch: agent/verifyskmana
worktree: ~/code/d2lod-wt/verifyskmana
skip_specs: true
owns:
  - openspec/changes/verifyskmana/**
  - docs/research/verify-skmana.md
  - tools/emu/vsm_*
exclusive: []
gates:
  - tools/ci/check.sh
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
merge: 3f14d22
```

- [x] 1 Read FUN_0056c160 and FUN_0056bfe0 and write an own oracle
- [x] 2 Run own harness vsm_diff.py against game.exe
- [x] 3 Write docs/research/verify-skmana.md
