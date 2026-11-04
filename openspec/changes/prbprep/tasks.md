# Tasks

```yaml
change: prbprep
status: in-progress
agent: prbprep
branch: agent/prbprep
worktree: ~/code/d2lod-wt/prbprep
skip_specs: true
owns:
  - openspec/changes/prbprep/**
  - docs/research/pr-batch-b-body-r5.md
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: Rebase and checks done locally on the worker host (Release build, six suites, presubmit --quick 0 FAIL, pr_guard 0 FAIL). Not verified live; the oracle run was not repeated. Nothing pushed or posted.
```

- [x] fetch upstream, rebase candidate onto upstream main
- [x] pr_guard, Release build, touched tests, presubmit --quick
- [x] write the updated body
