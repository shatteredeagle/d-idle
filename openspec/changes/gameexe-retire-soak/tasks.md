# Tasks

> status: REWRITTEN 2026-11-04 for ADR 0015: this is the Gate A card (acceptance on d2d plus G-D A:B), not a game.exe retirement. Card id and file names stay for tracking.

```yaml
change: gameexe-retire-soak
status: deferred
agent: gameexe-retire-soak
branch: agent/gameexe-retire-soak
worktree: ~/code/d2lod-wt/gameexe-retire-soak
owns:
  - docs/gameexe-retire.md
exclusive: []
gates:
  - "d2d 24 h run: fresh, uncheated, Act 1 cleared, writes after start 0 (Gate 3, needs user go)"
  - "G-D A:B verdict EQUIVALENT at 20 percent on XP/h and kills/h (6 game.exe + 6 d2d lanes, fresh clvl 1-6 cell)"
merge:
result:
```

## 1. Work

- [ ] 1.1 record the d2d 24 h run (Gate 3): fresh, uncheated, Andariel dead and Act 2 entered, writes after start 0
- [ ] 1.2 collect the game.exe oracle reference lanes (2 h / 8 h / 24 h) used by G-D; not a release gate
- [ ] 1.3 G-D A:B verdict at 20 percent; Gate A = 1.1 plus 1.3
- [x] ~~regression soak on Game.exe when original-game work resumes~~ (superseded by ADR 0015)
