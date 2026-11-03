# Tasks

```yaml
change: b2rebase
status: archived
agent: b2rebase
branch: agent/b2rebase
worktree: d2lod-wt/b2rebase
skip_specs: true
owns:
  - docs/research/pr-batch2-body.md
  - docs/research/pr-batch2-run.txt
  - tools/emu/b2r_*
  - openspec/changes/b2rebase/**
exclusive:
  - fork-branch:local/batch2-bundle-r2
gates:
  - "local/batch2-bundle-r2 is the four bundle commits on upstream main 552acc9a, same authors, same subjects, nothing pushed upstream"
  - "g++ and clang++ CI flags, the 13 suites, clang-tidy, pr_guard per commit, presubmit --quick, WSL lane and MSVC 14.51 pass on the new head"
  - "counted differentials re-run on the new head; changecheck 0 warnings, leakcheck 0 hits"
result: "docs only"
credit:
  component: Batch 2 bundle rebase onto upstream main 552acc9a
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: b6bd809
```

## 1. Work

- [x] 1.1 local/batch2-bundle-r2 on 552acc9a, pushed to the private remote
- [x] 1.2 PR 178 impact on our areas
- [x] 1.3 CI parity on the new head: warnings, 13 suites, clang-tidy, pr_guard, presubmit, MSVC 14.51, WSL lane
- [x] 1.4 counted differentials on the new head (mana, Teleport, damage roll)
- [x] 1.5 pr-batch2-body.md, pr-batch2-run.txt, merge-lines.txt, OWNERS lines
