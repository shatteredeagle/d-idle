# Tasks

```yaml
change: b2bundle
status: archived
agent: b2bundle
branch: agent/b2bundle
worktree: d2lod-wt/b2bundle
skip_specs: true
owns:
  - docs/research/pr-batch2-body.md
  - docs/research/pr-batch2-run.txt
  - tools/emu/b2b_*
  - openspec/changes/b2bundle/**
exclusive:
  - fork-branch:local/batch2-monsterdmg-r5
  - fork-branch:local/batch2-bundle
gates:
  - "r5 is one commit on 3e073888 with the verifier's two text conditions; bundle is 4 commits in the order minmana, Teleport D1, Teleport D2, damage roll"
  - "CI flags g++ and clang++, the 13 suites, clang-tidy, pr_guard per commit, presubmit --quick, MSVC 14.51 and the WSL lane all pass on the bundle head"
  - "counted differentials re-run on the bundle head; PR body carries the numbers; changecheck 0 warnings, leakcheck 0 hits"
result: "docs only"
credit:
  component: Batch 2 bundle (minmana floor, Teleport, monster damage roll)
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 48a8a28
```

## 1. Work

- [x] 1.1 local/batch2-monsterdmg-r5 from r4 (ponytail text, one commit), pushed to the private remote
- [x] 1.2 local/batch2-bundle on 3e073888 (PR 178 still open; throwaway merge-tree clean)
- [x] 1.3 CI parity on the bundle head: warnings, 13 suites, clang-tidy, pr_guard, presubmit, MSVC 14.51, WSL lane
- [x] 1.4 counted differentials on the bundle head (mana, Teleport, damage roll)
- [x] 1.5 docs/research/pr-batch2-body.md, merge-lines.txt, OWNERS lines
