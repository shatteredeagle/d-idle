# Tasks

```yaml
change: covmonai2
status: archived
agent: covmonai2
branch: agent/covmonai2
worktree: ~/code/d2lod-wt/covmonai2
skip_specs: true
owns:
  - docs/research/covmonai2-trace.md
  - docs/research/covmonai2-runs.txt
  - tools/emu/mai3_*
  - openspec/changes/covmonai2/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; coverage.py --check; tools/ci/check.sh; openspec validate --all --strict"
result: "docs/tools only; see the doc for tags and limits"
live: "docs only, nothing run live"
merge: 2c6af24
```

- [x] read the coverage docs and the earlier monster AI traces
- [x] run game.exe's think for the untested AIs (F1 to F4) against d2d mon_think
- [x] classify every mismatch (harness artifact, deviation, not isolated)
- [x] trace doc, ledger rows, regenerated coverage table
- [x] checks and merge lines
