# Tasks

```yaml
change: pubsplit
status: archived
agent: pubsplit
branch: agent/pubsplit
worktree: ~/code/d2lod-wt/pubsplit
owns:
  - docs/process/public-split.md
  - docs/process/public-manifest.json
  - docs/process/leakcheck-allow.txt
  - docs/process/ci-checks.md
  - cmd/leakcheck/**
  - README.md
  - tools/hooks/pre-push
  - .forgejo/workflows/ci.yml
  - openspec/changes/pubsplit/**
exclusive: []
gates:
  - "go test -p 1 -count=1 ./cmd/leakcheck passes"
  - "go run ./cmd/changecheck: 0 warnings"
  - "tools/openspec.sh validate --all --strict passes"
merge: a7fd025
result: "public/private split: cmd/leakcheck, manifest, public-split doc, README, pre-push hook and CI warn step; offline only"
```

## 1. Work

- [x] 1.1 cmd/leakcheck with tests, allow list, manifest
- [x] 1.2 Hook and CI wiring (warn only), ci-checks.md line
- [x] 1.3 README.md at the root
- [x] 1.4 docs/process/public-split.md with rules, mechanics, cut checklist, work packages
