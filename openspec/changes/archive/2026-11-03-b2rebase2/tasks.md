# Tasks

```yaml
change: b2rebase2
status: archived
agent: b2rebase2
branch: agent/b2rebase2
worktree: d2lod-wt/b2rebase2
skip_specs: true
owns:
  - docs/research/pr-batch2-body.md
  - docs/research/pr-batch2-body-split.md
  - docs/research/pr-batch2-run.txt
  - docs/research/pr-upstream-rules-20261103.md
  - tools/emu/b2r3_*
  - openspec/changes/b2rebase2/**
exclusive:
  - fork-branch:local/batch2-bundle-r3
gates:
  - "local/batch2-bundle-r3 is the four bundle commits on upstream main 005c7cab, same authors, subjects, range-diff equal, nothing pushed upstream"
  - "changecheck 0 warnings, leakcheck 0 hits"
result: "docs and tools only; see card result"
credit:
  component: Batch 2 bundle onto upstream main 005c7cab, split proposal
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: f498e13
```

## 1. Work

- [x] 1.1 read the new upstream rules, summary in pr-upstream-rules-20261103.md
- [x] 1.2 local/batch2-bundle-r3 on 005c7cab (no conflict, range-diff equal), pushed to the private remote
- [x] 1.3 split branches local/batch2-mana-teleport and local/batch2-monsterdmg-pr, pushed
- [x] 1.4 parity, pr_guard, presubmit --quick, mana and Teleport seed 1 differentials
- [ ] 1.5 Teleport seed 9, damage sets, WSL lane, MSVC 14.51 (blocked: gate run on the worker host, build machine window; checklist in the run record)
- [x] 1.6 bodies on the new template, budget, merge-lines.txt, OWNERS lines
