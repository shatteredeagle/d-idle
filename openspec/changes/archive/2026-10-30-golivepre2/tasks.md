# Tasks

```yaml
change: golivepre2
status: archived
agent: GOLIVEPREP2
branch: agent/golivepre2
worktree: ~/code/d2lod-wt/golivepre2
owns:
  - cmd/changecheck/**
  - tools/hooks/**
  - .github/workflows/ci.yml
  - docs/ci.md
  - docs/process/ci-checks.md
  - docs/process/hooks.md
  - docs/plan/STATUS.md
  - openspec/changes/golivepre2/**
exclusive: []
gates:
  - "go test -p 1 ./cmd/changecheck ./cmd/leakcheck passes (TestDefaultBranch*)"
  - "sh tools/hooks/test-pre-push.sh passes (main, master, other branch, failing check, bypass)"
  - "leakcheck fail mode exit 0; changecheck 0 warnings; tools/openspec.sh validate --all --strict passes"
merge: 4580faeb
result: "go-live prep 2 for the existing public repo shatteredeagle/d-idle: changecheck/pre-push branch-agnostic (origin/HEAD, main, master); CI only on main/tags/PRs; ops: D2_PAGES_BRANCH publisher setting, tools/golive-pages.sh (plan/execute/publisher/push-main/rollback, GOLIVE_CONFIRM guard, dry-run), runbook docs/process/go-live-cut.md rewritten for the gh-pages plan"
```

## 1. Work

- [x] 1.1 changecheck default-branch detection and tests
- [x] 1.2 pre-push hook for main and master, hook test script
- [x] 1.3 ci.yml note, docs
