# Tasks

```yaml
change: roadmap
status: archived
agent: roadmap
branch: agent/roadmap
worktree: ~/code/d2lod-wt/roadmap
owns:
  - docs/roadmap.md
  - docs/process/dispatcher-brief-proposed.md
  - docs/BACKLOG.md
  - openspec/changes/m1-audittests/**
  - openspec/changes/m1-stashcap/**
  - openspec/changes/m1-trips/**
  - openspec/changes/m1-liveproof/**
  - openspec/changes/m2-roam/**
  - openspec/changes/m2-tripwp/**
  - openspec/changes/m2-abdesign/**
  - openspec/changes/m3-andariel/**
  - openspec/changes/m3-merc/**
  - openspec/changes/m3-routeguard/**
  - openspec/changes/m3-act1chain/**
  - openspec/changes/m3-act1run/**
  - openspec/changes/m1-loot/**
  - openspec/changes/m2-soakexit/**
  - openspec/changes/m2-earlybase/**
exclusive: []
gates:
  - "go run ./cmd/changecheck -cli has 0 warnings after `changecheck status -write`"
  - "go run ./cmd/leakcheck -mode fail finds 0"
  - "tools/openspec.sh validate --all --strict passes including every new card and delta"
  - "docs/roadmap.md is at most 150 lines and has an Alignment section naming every SME disagreement"
merge: 51cad66
result: "Docs and cards only, nothing run live: docs/roadmap.md, 14 milestone cards (4 proposed, 10 deferred) with deltas on town-basics, act1-campaign, fidelity-gate, early-game-policy, merc-policy; dispatcher brief proposal; SME alignment recorded."
```

## 1. Work

- [x] 1.1 Diagnosis from git log, the archive and the docs (counts by theme) in docs/roadmap.md
- [x] 1.2 Milestones with exit criteria tied to the specs; first draft reviewed by d2-sme and spec-sme; alignment recorded
- [x] 1.3 Cards m1-*, m2-*, m3-* with deltas; docs/OWNERS lines for the proposed ones; validate and changecheck clean
- [x] 1.4 docs/process/dispatcher-brief-proposed.md; BACKLOG pointer line; report with decisions and proposed ADR text
