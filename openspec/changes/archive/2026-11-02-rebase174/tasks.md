# Tasks

```yaml
change: rebase174
status: archived
agent: rebase174
branch: agent/rebase174
worktree: d2lod-wt/rebase174
skip_specs: true
owns:
  - docs/research/rebase174-20261102.md
  - docs/research/pr-freeze-rules-draft.md
  - docs/research/pr-vendor-repair-gate-draft.md
  - docs/research/pr-bundle-1-draft.md
  - openspec/changes/rebase174/**
exclusive:
  - fork-branch:local/freeze-rules-v5
  - fork-branch:local/vendor-repair-gate-v4
  - fork-branch:local/skmelee3-d1-r174
  - fork-branch:local/skmelee3-d2-r174
  - fork-branch:local/bundle-1
gates:
  - "seven suites green on each new head; counted differential base vs head on the new upstream main"
result: "docs only"
live: "offline: no game; game.exe's own functions run under the emulator on the worker host"
merge: 436019b
credit:
  component: Bundle 1 rebase onto upstream main
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifybundle, independent run)"]
  consent: none
```

- [x] 1 Rebase the four candidates onto the new upstream main (D2 conflict resolved)
- [x] 2 Seven suites and the counted differentials, base vs head on the new main
- [x] 3 CI parity (g++, clang++, tidy, pr_guard, presubmit incl. the attribution check)
- [x] 4 Push the new branches to the private remote only
- [x] 5 Table doc and draft base lines, changecheck, leakcheck
- [x] 6 Bundle branch local/bundle-1, checks on its head, combined draft
