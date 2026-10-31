# Tasks

```yaml
change: roadmap
status: in-progress
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
merge:
result:
```

## 1. Work

- [ ] 1.1 Diagnosis from git log, the archive and the docs (counts by theme) in docs/roadmap.md
- [ ] 1.2 Milestones with exit criteria tied to the specs; first draft reviewed by d2-sme and spec-sme; alignment recorded
- [ ] 1.3 Cards m1-*, m2-*, m3-* with deltas; docs/OWNERS lines for the proposed ones; validate and changecheck clean
- [ ] 1.4 docs/process/dispatcher-brief-proposed.md; BACKLOG pointer line; report with decisions and proposed ADR text
