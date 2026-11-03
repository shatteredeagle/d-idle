# Tasks

```yaml
change: prbuildb2
status: archived
agent: prbuildb2
branch: agent/prbuildb2
worktree: d2lod-wt/prbuildb2
skip_specs: true
owns:
  - docs/research/pr-batch2-draft.md
  - tools/emu/pb2_mana.sh
  - tools/emu/pb2_tele.sh
  - openspec/changes/prbuildb2/**
exclusive:
  - fork-branch:local/batch2-minmana
  - fork-branch:local/batch2-teleport
gates:
  - "counted differential base vs head on a real d2d build >= 20,000 cases per part; seven suites plus test_sk_ground green; CI parity"
merge: 4137294
result: "docs only; fork branches on the private remote"
credit:
  component: Upstream batch 2 candidates (minmana floor, Teleport)
  author: owner
  agents: [Claude Code]
  consent: none
```

## 1. Work

- [x] 1.1 Fetch upstream read-only, base both branches on upstream main after #177
- [x] 1.2 local/batch2-minmana: the mana check without the floor, the payment rule, counted differential
- [x] 1.3 local/batch2-teleport: room limit (D1) and nearest free spot (D2), a counted differential each
- [x] 1.4 CI parity (g++, clang++, extra warnings, tidy, pr_guard, presubmit, suites)
- [x] 1.5 Throwaway merge of each branch with #178's head, conflicts reported
- [x] 1.6 Push both branches to the private remote only
- [x] 1.7 docs/research/pr-batch2-draft.md, changecheck, leakcheck
