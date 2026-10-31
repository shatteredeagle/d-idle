# Tasks

```yaml
change: upstream-bundle
status: in-progress
agent: bundlepr
branch: agent/bundlepr
worktree: ~/code/d2lod-wt/bundlepr
owns:
  - docs/process/upstream-pr-standard.md
  - docs/research/pr-bundle-draft.md
  - docs/research/pr-bundle-evidence-pack.md
  - docs/research/README.md
  - docs/BACKLOG.md
  - openspec/changes/upstream-bundle/**
exclusive: []
gates:
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result: "docs and spec only; bundle branch prepared locally on the worker host, nothing pushed or posted"
```

## 1. Work

- [x] 1.1 Standard: claims per commit, bundles allowed, spacing replaced
- [x] 1.2 Spec delta for the two requirements
- [x] 1.3 Bundle draft and evidence pack
- [x] 1.4 BACKLOG line, research index, changecheck, leakcheck, validate
