# Tasks

```yaml
change: spec-process
status: in-progress
agent: specfix
branch: agent/specfix
worktree: ~/code/d2lod-wt/specfix
owns:
  - openspec/changes/spec-process/**
  - openspec/README.md
  - docs/research/pr-get-hit-rule-evidence-pack.md
  - docs/research/pr-missile-damage-fine-evidence-pack.md
  - docs/research/pr-get-hit-length-evidence-pack.md
  - openspec/changes/archive/2026-10-31-hitlen/.openspec.yaml
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
  - "every posted: line was checked with a read-only gh api GET"
merge:
result: "README rules for skip_specs, unchecked non-merge commits, archive prefix and propose-time deltas; upstream-pr delta for the posted: log; three posted: backfills; hitlen .openspec.yaml. Docs only."
```

## 1. Work

- [x] 1.1 openspec/README.md: When skip_specs is allowed; commits without a card; archive date; deltas from propose
- [x] 1.2 Delta ADDED requirement in upstream-pr (posted: log line)
- [x] 1.3 Backfill posted: lines for PR 134, 135, 136, each checked with gh api GET
- [x] 1.4 .openspec.yaml for archive/2026-10-31-hitlen
