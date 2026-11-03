# Tasks

```yaml
change: cport
status: archived
agent: cport
branch: agent/cport
worktree: d2lod-wt/cport
skip_specs: true
owns:
  - docs/research/pr-batch2-C-rows.md
  - openspec/changes/cport/**
exclusive:
  - fork-branch:local/batch2-C-r1
gates:
  - "C commits: one claim each with its test and oracle, numeric noreply identity, private remote only"
  - "changecheck 0 new warnings, leakcheck 0 hits, merge_lines_check ok"
result: "docs only in this repo; fork branch built and measured on the Windows build worker, not in a live game"
credit:
  component: mana payment on the action frame, action-frame tick and armor state group as commits on batch A
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 99328d5
```

## 1. Work

- [x] 1.1 read the maintainer rules, build local/batch2-C-r1 from batch A (3 commits), resolve conflicts
- [x] 1.2 pr_guard, presubmit --quick, ruff
- [x] 1.3 counted differentials on the exact head, parity lanes
- [x] 1.4 docs/research/pr-batch2-C-rows.md, merge-lines.txt, OWNERS lines
