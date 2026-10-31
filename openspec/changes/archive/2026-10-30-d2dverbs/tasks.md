# Tasks

```yaml
change: d2dverbs
status: archived
agent: d2dverbs
branch: agent/d2dverbs
worktree: ~/code/d2lod-wt/d2dverbs
owns:
  - docs/d2d-lab.md
  - docs/d2d-fork.md
exclusive:
  - d2d-lab-binary-lab3
  - fork-branch:lab3
gates:
  - "each verb verified in a private run before the docs call it V"
  - "start.sh is not pointed at lab3 until verified end to end"
merge: a576af7
result: "lab3 = lab2 + cmd belt put, dump ui rects, debug wp; pack coin/stash coin/grid/belt rects verified, store/waypoint/gold rects and portal/chest interact NOT verified; repair constant 368,461 is the repair-all button"
```

## 1. Work

- [x] done belt put / tobelt verb
- [x] done dump ui rects
- [x] done object / portal use and debug wp (map to existing verbs where they exist)
- [x] done build d2d.lab3, verify privately, push lab3 to the forge
- [x] done docs
