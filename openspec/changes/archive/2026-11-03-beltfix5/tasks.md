# Tasks

```yaml
change: beltfix5
status: archived
agent: beltfix5
branch: agent/beltfix5
worktree: ~/code/d2lod-wt/beltfix5
skip_specs: true
owns:
  - internal/town/beltfix5_test.go
  - internal/bot/beltfix5_test.go
  - docs/research/beltfix5.md
  - openspec/changes/beltfix5/**
exclusive: []
gates:
  - "go test -p 1 ./internal/bot ./internal/town; whole repo vet+test on the worker host; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh"
result: "whole repo vet and test pass on the worker host; replay test of the window"
credit:
  author: owner
  agents: [Claude Code]
merge: 4f7c85e
```

- [x] 1 reconstruct the 0338 window from the minute lines and the bot log (docs/research/beltfix5.md)
- [x] 2 town withdraw chore (TestWithdrawForTheKitWhenNoHealingPotion fails when the chore is off)
- [x] 3 bot counts pack plus stash gold with no healing potion (TestBankedGoldCountsWithNoHealingPotion fails when reverted)
- [x] 4 belt-window numbers of the four Gate 1 runs of 2026-11-02/03 (section 5)
- [ ] 5 live check (section 6)
