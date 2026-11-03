# Tasks

```yaml
change: prbuildground
status: archived
agent: prbuildground
branch: agent/prbuildground
worktree: d2lod-wt/prbuildground
skip_specs: true
owns:
  - docs/research/pr-ground-burst-draft.md
  - openspec/changes/prbuildground/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "draft only; bundle local/ground-burst-v1 built and measured offline in the private fork, not verified live (no game)"
live: "no game, offline"
merge: 45fa51c
```

- [x] inventory proven versus read-only ground and burst deviations
- [x] build the bundle on upstream main, counted differentials base versus head
- [x] CI parity, suites, presubmit
- [x] draft docs/research/pr-ground-burst-draft.md
