# Tasks

```yaml
change: golivepre
status: archived
agent: GOLIVEPREP
branch: agent/golivepre
worktree: ~/code/d2lod-wt/golivepre
owns:
  - cmd/changecheck/**
  - .github/**
  - README.md
  - tools/ops.sh
  - .gitignore
  - docs/process/ci-checks.md
  - .claude/settings.json
  - openspec/changes/golivepre/**
exclusive: []
gates:
  - "go test -p 1 ./cmd/changecheck ./cmd/leakcheck passes (history_test.go: one commit, merge history, baseline, shallow clone)"
  - "go run ./cmd/changecheck -cli has 0 warnings on this tree and passes on a one-commit export (ops tools/golive-dry.sh)"
  - "go run ./cmd/leakcheck -mode fail exits 0; tools/openspec.sh validate --all --strict passes"
merge: c463088
result: go-live prep: changecheck tolerates shallow/single-commit history (INFO, exit 0; history_baseline supported); public GitHub CI workflow (.github/workflows/ci.yml: checks, test, release on v* tags with GITHUB_TOKEN; official actions only; unvalidated by a tool until the first Actions run); README history/Go notes; .claude/settings.json moved to the private overlay; ops repo has docs/process/go-live-cut.md and tools/golive-dry.sh (dry run green)
```

## 1. Work

- [x] 1.1 changecheck tolerates a history-less or shallow repo (INFO, exit 0) and supports history_baseline; unit tests
- [x] 1.2 .github/workflows/ci.yml for the public repo
- [x] 1.3 README: history sentence, Go version, contributing and CI pointers
- [x] 1.4 .claude/settings.json moves to the ops overlay (tools/ops.sh link, .gitignore)
- [x] 1.5 ops repo: docs/process/go-live-cut.md and tools/golive-dry.sh (committed there, not in this repo)
