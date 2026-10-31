# Tasks

```yaml
change: prmeasure
status: archived
merge: 2f93492
agent: prmeasure
branch: agent/prmeasure
worktree: ~/code/d2lod-wt/prmeasure
owns:
  - docs/research/pr-missile-damage-fine-measured.md
  - docs/probes/prmeasure.csv
  - tools/prmeasure/**
exclusive:
  - d2d-lab-binary
gates:
  - A/B run complete, nothing pushed
result: 'missile PR corroboration: CB8 -13 % casts per kill, CB4 no change, CB1 unusable (load noise); the PR proof is the differential, not this'
```

- [x] A/B runs (24, 3 slots)
- [x] control build (unpatched, same flags) to separate patch from build effect
- [x] doc
