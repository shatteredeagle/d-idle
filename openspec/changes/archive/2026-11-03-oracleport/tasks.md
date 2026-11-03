# Tasks

```yaml
change: oracleport
status: archived
agent: oracleport
branch: agent/oracleport
worktree: d2lod-wt/oracleport
skip_specs: true
owns:
  - openspec/changes/oracleport/**
  - docs/research/pr-batch2-A-body.md
  - docs/research/pr-batch2-B-body.md
exclusive:
  - "fork-branch:local/batch2-A-r5"
  - "fork-branch:local/batch2-B-r2"
gates:
  - "changecheck 0 new warnings; leakcheck -mode fail 0; merge_lines_check ok"
  - "pr_guard 0 FAIL and presubmit --quick 0 FAIL on each fork branch"
credit:
  component: BATCH2
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
result: "docs only"
merge: eda9c72
```

## 1. Work

- [x] 1.1 Read the maintainer's AGENTS.md, CONTRIBUTING.md, PR template, pr_guard.py, presubmit.py, tools/emu README and review checklist at upstream main
- [x] 1.2 Audit batch A against them; new branch local/batch2-A-r5, pushed to the private remote
- [x] 1.3 Port batch B with its oracle; new branch local/batch2-B-r2, pushed to the private remote
- [x] 1.4 Re-run on the Windows build worker (unit tests with game data, the counted differentials, GCC and tidy through presubmit)
- [x] 1.5 PR bodies A and B, card, OWNERS, merge-lines
