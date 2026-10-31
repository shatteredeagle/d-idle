# Tasks

```yaml
change: d2dcast
status: archived
agent: d2dcast
branch: agent/d2dcast
worktree: ~/code/d2lod-wt/d2dcast
owns:
  - docs/d2d-upstream-cast.md
exclusive:
  - fork-branch:upstream-cast
  - upstream-pr:cast
gates:
  - "cast period measured against the original game or its documented frame table before any claim (V-d2d only is not enough)"
  - "nothing is posted to GitHub without the user go"
merge: ca1916e
result: "superseded by casttrace (docs/research/cast-trace-gameexe.md) and PR #128; fork branch not prepared here (not verified live)"
```

## 1. Work

- [ ] 1.1 Reproduce the 14-tick cast period in the lab
- [ ] 1.2 Prepare the patch on its own fork branch and push it to <owner>/d2d-fork
- [ ] 1.3 Draft PR text for review, post nothing
