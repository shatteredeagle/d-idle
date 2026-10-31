# Tasks

```yaml
change: decompxp
status: archived
agent: DECOMPXP
branch: agent/decompxp
worktree: ~/code/d2lod-wt/decompxp
owns:
  - docs/research/xp-award-trace.md
  - docs/research/upstream-pr-candidates-xp.md
  - tools/emu/xp_*.py
exclusive: []
gates:
  - "every address cited is read in game.exe 1.14d or run in the oracle (V) or tagged S/U"
  - "nothing pushed, posted or filed on GitHub"
merge: 6a63a52
result: "XP award formula traced (FUN_0057e990/0057e480/0057e2f0/0057e390/0057e510); per-monster XP identical on both backends clvl 1-7; Den +60 % and Blood Moor +20 % = composition + live kill counting; d2d kill_exp bug from diff 6 up fixed upstream 01337ba3 (lab branch must rebase: forkrebase). Merged 6a63a52"
```

## 1. Work

- [x] 1.1 trace the award path (FUN_0057e990 -> FUN_0057e480 -> FUN_0057e2f0 / FUN_0057e390 / FUN_0057e510)
- [x] 1.2 per-monster expected XP for the Den and Blood Moor vs d2d and live
- [x] 1.3 d2d kill_exp differences, oracle grid vs d2d
- [x] 1.4 PR-ready branch under ~/d2d-lab/pr/ and draft body (not posted)
