# Tasks

```yaml
change: covmissiles
status: archived
agent: covmissiles
branch: agent/covmissiles
worktree: ~/code/d2lod-wt/covmissiles
skip_specs: true
owns:
  - docs/research/covmissiles-trace.md
  - tools/emu/cm_*
  - openspec/changes/covmissiles/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; coverage.py --check; tools/ci/check.sh; openspec validate --all --strict"
result: "docs only; see the doc for tags and limits"
live: "docs only, nothing run live"
merge: 975bfbd
```

- [x] read the coverage docs and the closest traces
- [x] hit-roll differential for the Sorceress skills not yet run (cm_run.sh hit)
- [x] table and flight differential for all 684 rows (cm_run.sh flight)
- [x] read d2d's loader and column use (cm_sweep.sh)
- [x] trace doc, ledger rows, regenerated coverage table
- [x] checks and merge lines
