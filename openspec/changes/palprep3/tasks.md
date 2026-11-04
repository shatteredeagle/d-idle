# Tasks

```yaml
change: palprep3
status: in-progress
agent: palprep3
branch: agent/palprep3
worktree: ~/code/d2lod-wt/palprep3
skip_specs: true
owns:
  - openspec/changes/palprep3/**
  - docs/research/pr-palskill-body-r3.md
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "local only: PAL D2/D3 cherry-picked onto upstream 7591a92e, no conflict (local/pal-d2d3-r3 1f91a216, pushed to the private fork remote only); verifier caveats applied; oracle 0 differ on both harnesses; body r3 written; nothing pushed to GitHub or posted; not verified live"
```

- [x] fetch upstream, cherry-pick the two commits, apply the verifier caveats
- [x] pr_guard, Release build, touched tests, clang-tidy, presubmit --quick
- [x] rerun the oracle comparisons (verifier seeds and the author's) on the new head
- [x] write body r3
