# Tasks

```yaml
change: specdocs
status: archived
agent: specdocs
branch: agent/specdocs
worktree: ~/code/d2lod-wt/specdocs
owns:
  - openspec/changes/specdocs/**
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "go run ./cmd/changecheck -cli prints 0 warnings"
  - "go run ./cmd/leakcheck -mode fail reports no hit in openspec/changes/specdocs"
merge: 02a94eb
result: 'four OpenSpec specs for the doc-only rules: upstream-pr (13), process-hygiene (9), public-split (10), licence-policy (7)'
```

Spec-only card; the archive step applies the four deltas to openspec/specs.

## 1. Specs

- [x] 1.1 specs/upstream-pr/spec.md (ADDED delta); validate --strict passes
- [x] 1.2 specs/process-hygiene/spec.md (ADDED delta); validate --strict passes
- [x] 1.3 specs/public-split/spec.md (ADDED delta); validate --strict passes
- [x] 1.4 specs/licence-policy/spec.md (ADDED delta); validate --strict passes

## 2. Review

- [x] 2.1 spec-sme reviews the form; findings applied (splits, untestable scenarios reworded) or listed in the report; verified by validate --strict and changecheck 0 warnings
