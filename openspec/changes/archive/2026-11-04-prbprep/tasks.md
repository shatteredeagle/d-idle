# Tasks

```yaml
change: prbprep
status: archived
agent: prbprep
branch: agent/prbprep
worktree: ~/code/d2lod-wt/prbprep
skip_specs: true
owns:
  - openspec/changes/prbprep/**
  - docs/research/pr-batch-b-body-r5.md
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "local only: nothing pushed upstream; body awaits the go"
merge: df6e379
```

- [x] fetch upstream, rebase candidate onto upstream main
- [x] pr_guard, Release build, touched tests, presubmit --quick
- [x] write the updated body
