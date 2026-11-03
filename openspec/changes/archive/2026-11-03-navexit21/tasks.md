# Tasks

```yaml
change: navexit21
status: archived
agent: navexit21
branch: agent/navexit21
worktree: ~/code/d2lod-wt/navexit21
skip_specs: true
owns:
  - internal/d2dworld/world.go
  - internal/d2dworld/world_test.go
  - internal/town/shop.go
  - internal/town/chores.go
  - internal/town/navexit21_test.go
  - docs/navexit21.md
  - openspec/changes/navexit21/**
exclusive: []
gates:
  - "go test -p 1 ./internal/d2dworld ./internal/town ./internal/nav passes (town needs the extracted tables)"
  - "Gate 1 seed 17 re-run: no funded belt window over 5 virtual minutes, no no-kill over 10 (live, not done)"
result: "Offline only, not verified live. Fix 1 d2dworld keepBorders (TestBordersSurviveAnEmptyProbeRound fails without it); fix 2 town buyFirstTP for an empty tome (TestEmptyTomeGetsAScrollBeforeThePotionsEatTheGold). Why a probe round came back empty in the real game is U."
credit:
  author: owner
  agents: [Claude Code]
merge: bf29e35
```

## 1. Work

- [x] 1.1 Find why the exit from level 2 to level 1 is missing (docs/navexit21.md)
- [x] 1.2 keepBorders fix and test
- [x] 1.3 Empty-tome first scroll and test
- [ ] 1.4 Gate 1 seed 17 re-run (live)
