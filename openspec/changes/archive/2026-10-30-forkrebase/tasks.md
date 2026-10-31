# Tasks

```yaml
change: forkrebase
status: archived
agent: forkrebase
branch: agent/forkrebase
worktree: ~/code/d2lod-wt/forkrebase
owns:
  - docs/d2d-lab6.md
  - docs/d2d-fork.md
exclusive:
  - d2d-lab-binary
  - fork-branch:lab6
gates:
  - "d2d.lab6 gives identical det.py hashes 3/3 and upstream tests pass"
merge: d5dadf4
result: lab6 pushed to the forge e417d2b8; tests, det hashes and speed verified on the worker host (hidden d2d, slot wrapper); not verified with a live game, screenshot verb not re-run
```

## 1. Work

- [x] 1.1 inventory the lab5 patch set
- [x] 1.2 lab6 branch on current upstream main (price and cast patches dropped)
- [x] 1.3 build and verify d2d.lab6
- [x] 1.4 push lab6 to the forge, docs
- [x] 1.5 coordinator archives the card at merge
