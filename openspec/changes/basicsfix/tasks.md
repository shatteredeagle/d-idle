# Tasks

```yaml
change: basicsfix
status: in-progress
agent: basicsfix
branch: agent/basicsfix
worktree: ~/code/d2lod-wt/basicsfix
skip_specs: true
owns:
  - openspec/changes/basicsfix/**
  - internal/town/basicsfix.go
  - internal/town/basicsfix_test.go
  - internal/town/basicsaudit_test.go
  - internal/town/stash.go
  - internal/town/invstash_test.go
  - internal/town/seed_test.go
  - docs/basics-audit.md
exclusive: []
gates:
  - "go test -short -p 4 ./... and go test -race ./internal/town ./internal/bot pass on the worker host"
  - "tools/openspec.sh validate --all --strict passes"
result: ""
```

- [x] 1 ID scrolls only for likely valuables (idScrollNeed)
- [ ] 2 second sell-trip door in bot/goal.go: SKIPPED, same line as XPGAP's edit; after it merges apply povertySellOK to the else-if
- [x] 3 failed repair latch (repairLatch)
- [x] 4 bank before shop (bankFirst)
- [x] 5 potions first (potionsFirst, tpFirstLevel)
