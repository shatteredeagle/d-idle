# Tasks

```yaml
change: killfix
status: archived
agent: killfix
branch: agent/killfix
worktree: ~/code/d2lod-wt/killfix
owns:
  - docs/killfix.md
  - internal/bot/emptyarea_test.go
  - tools/d2dloop/pm.sh
  - tools/d2dloop/kfbuild.sh
exclusive: []
gates:
  - "before/after kills/h and XP/h on the same 4 seeds"
merge: d6cd0d9
result: verified on d2d (4 seeds x 2 virtual h, 8 runs per arm for baseline and final): empty-area leave, mouthStep flood and combat mouth guard give no kills/h or XP/h gain (164 vs 154 kills/h, 3830 vs 3480 XP/h, noise about 10 %); the guard cuts border crossings 28.5 to 20.8. All ship off or inert; numbers in docs/killfix.md. Game.exe-affecting (needs G1/G2 check): nav.mouthStep flood fallback only. Not verified on Game.exe.
```

## 1. Work

- [x] 1.1 baseline on 4 seeds in parallel slots
- [x] 1.2 empty-area leave (EmptyAreaSec)
- [x] 1.3 mouthStep flood fallback
- [x] 1.4 re-measure and document
