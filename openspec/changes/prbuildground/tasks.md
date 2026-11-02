# Tasks

```yaml
change: prbuildground
status: in-progress
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
result: "draft only; bundle built and measured in the private fork on the worker host, not live"
live: "no game, offline"
merge: ""
```

- [ ] inventory proven versus read-only ground and burst deviations
- [ ] build the bundle on upstream main, counted differentials base versus head
- [ ] CI parity, suites, presubmit
- [ ] draft docs/research/pr-ground-burst-draft.md
