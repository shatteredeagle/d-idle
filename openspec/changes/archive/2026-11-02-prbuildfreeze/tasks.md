# Tasks

```yaml
change: prbuildfreeze
status: archived
agent: prbuildfreeze
branch: agent/prbuildfreeze
worktree: d2lod-wt/prbuildfreeze
skip_specs: true
owns:
  - openspec/changes/prbuildfreeze/**
  - docs/research/pr-freeze-rules-draft.md
exclusive:
  - fork-branch:local/freeze-rules-v4
gates:
  - "changecheck 0 warnings; leakcheck -mode fail 0"
result: "docs only; PR hold in force"
merge: e9f2394
credit:
  component: Freeze rules
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifyfreeze, independent run; freeze re-run in card verifybundle)"]
  consent: none
```

## 1. Work

- [x] 1.1 Order versus PR 159 (merged meanwhile; v3 and v4 shapes, throwaway merges)
- [x] 1.2 Branch local/freeze-rules-v3 on upstream main, pushed to the private remote
- [x] 1.3 Rebuild, suites, freeze differential on the final head
- [x] 1.4 CI parity (g++, clang++, include-cleaner, pr_guard, presubmit --quick)
- [x] 1.5 Draft docs/research/pr-freeze-rules-draft.md
