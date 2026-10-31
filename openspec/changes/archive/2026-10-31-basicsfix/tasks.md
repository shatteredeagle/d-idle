# Tasks

```yaml
change: basicsfix
status: archived
agent: basicsfix
branch: agent/basicsfix
worktree: ~/code/d2lod-wt/basicsfix
skip_specs: false
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
result: "Offline only (sim rig), not verified live. XPGAP follow-ups (sell door, no-scroll trip reason) applied at merge (c50a7a1, 6b9f52f)."
merge: 478499f
```

- [x] 1 ID scrolls only for likely valuables (idScrollNeed)
- [x] 2 second sell-trip door in bot/goal.go: applied at merge (lowGoldSellOK, povertySellMin)
- [x] 3 failed repair latch (repairLatch)
- [x] 4 bank before shop (bankFirst)
- [x] 5 potions first (potionsFirst, tpFirstLevel)
