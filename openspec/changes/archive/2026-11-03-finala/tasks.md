# Tasks

```yaml
change: finala
status: archived
agent: finala
branch: agent/finala
worktree: d2lod-wt/finala
skip_specs: true
owns:
  - openspec/changes/finala/**
  - docs/research/pr-guardfix-body.md
  - docs/research/pr-batch2-A-final-body.md
exclusive:
  - "fork-branch:local/guardfix-3"
  - "fork-branch:local/batch2-A-r9"
gates:
  - "changecheck 0 new warnings; leakcheck -mode fail 0; merge_lines_check ok"
  - "fixed pr_guard 0 FAIL on local/batch2-A-r9 and on local/guardfix-3; the new add-and-cite test fails on upstream's guard"
credit:
  component: GUARDFIX
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
result: "private fork branches local/guardfix-3 (93851bef) and local/batch2-A-r9 (57d7c39b) on upstream main 9785632c, pushed to the private and public fork; verified on the Windows build worker (not a live game); PR texts drafted, nothing posted"
merge: 0ba9a03
```

## 1. Work

- [x] 1.1 local/guardfix-3: one commit on upstream main 9785632c, the pr_guard cited-tool check on the PR head, with two tests
- [x] 1.2 local/batch2-A-r9: the three batch-A commits without the guard commit, on upstream main 9785632c, one fixup for ruff B905
- [x] 1.3 Verify the final heads on the Windows build worker (counted differentials, parity lane, MSVC 14.51, guard run as CI does)
- [x] 1.4 Push both branches to the private remote and the public fork (new branch names only, no force-push)
- [x] 1.5 PR texts for the guard fix and for batch A
