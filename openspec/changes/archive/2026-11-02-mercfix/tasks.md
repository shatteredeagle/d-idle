# Tasks

```yaml
change: mercfix
status: archived
agent: mercfix
branch: agent/mercfix
worktree: d2lod-wt/mercfix
skip_specs: true
owns:
  - cmd/d2bot/mercwire.go
  - cmd/d2bot/mercwire_test.go
  - cmd/d2bot/d2dwire.go
  - docs/botlogs.md
  - openspec/changes/mercfix/**
exclusive: []
gates:
  - "go test -p 1 ./cmd/d2bot ./internal/merc ./internal/d2dmerc ./internal/task; changecheck 0 warnings; leakcheck 0 hits"
result: "go tests cmd/d2bot internal/merc d2dmerc task pass; not verified live"
live: "not run"
credit:
  component: mercfix
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card mercfix, own reading of the Gate 2 log)"]
  consent: none
merge: cde0413
```

- [x] Reconstruct the claim timeline from the Gate 2 bot log
- [x] Find why the claim and the guard disagree (hook nil with the policy off)
- [x] guardMercHook and claimDep read-back with tests
- [x] docs/botlogs.md line, card, shared-file lines
