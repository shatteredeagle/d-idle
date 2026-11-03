# Tasks

```yaml
change: skground2
status: archived
agent: skground2
branch: agent/skground2
worktree: ~/code/d2lod-wt/skground2
skip_specs: true
owns:
  - docs/research/sk-ground2-trace.md
  - docs/research/pr-blizzard-draft.md
  - tools/emu/sk_gnd2_*
  - openspec/changes/skground2/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "Fire Wall matched game.exe subtiles on the real build (80 of 80), Hydra read only, shard hit area and offset deviations measured on the real build; local PR candidate for the offsets awaits an independent verifier; not verified live"
live: "docs only, nothing run live"
merge: 6e7b133
credit:
  component: Standing missile footprint and burst roll (ground)
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifybundle, own harness)"]
  consent: none
```

- [x] Fire Wall (do 24, maker do 6, hits) read and run against a real d2d build
- [x] Hydra start check and do 144 read
- [x] D4 oracle (game.exe unit search) and d2d real-build measurement
- [x] D1 real-build differential, local PR candidate, pr_guard
- [x] trace doc, PR draft, matrix cells, drill recipe, shared-file lines
