# Tasks

```yaml
change: d2dprice
status: archived
agent: d2dprice
branch: agent/d2dprice
worktree: ~/code/d2lod-wt/d2dprice
owns:
  - docs/d2d-upstream-price.md
exclusive:
  - fork-branch:upstream-price
  - upstream-pr:price
gates:
  - "Tome of Town Portal price matches the original (450 live vs 250 on d2d) with the source of the number tagged"
  - "nothing is posted to GitHub without the user go"
merge: 9041c9e
result: "PR submitted 2026-10-30: https://github.com/psi29a/d2decomped/pull/127 (commit 7ae861d, static reading of game.exe plus live prices; not covered: keys/javelins ponytail)"
```

## 1. Work

- [x] 1.1 Patch and test in the lab
- [x] 1.2 Push the branch to <owner>/d2d-fork
- [x] 1.3 Draft PR text, post nothing
