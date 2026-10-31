# Tasks

```yaml
change: grillpane
status: archived
agent: grillpane
branch: agent/grillpane
worktree: ~/code/d2lod-wt/grillpane
owns:
  - docs/process/mini-grill.md
  - tools/grill.sh
  - docs/process/grill-workflow.md
exclusive: []
gates:
  - "sh -n tools/grill.sh passes and --print-only prints both modes"
merge: 5d2a999
result: "mini-grill protocol; tools/grill.sh --pane via the head box (split/run verified, full path untested)"
```

## 1. Work

- [x] 1.1 mini-grill.md protocol
- [x] 1.2 grill.sh --pane / --here
- [x] 1.3 grill-workflow.md update
