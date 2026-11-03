# Tasks

```yaml
change: branchclean
status: archived
agent: branchclean
branch: agent/branchclean
worktree: d2lod-wt/branchclean
skip_specs: true
owns:
  - docs/research/branch-cleanup-plan.md
  - openspec/changes/branchclean/**
exclusive: []
gates: merge_lines_check --changed; changecheck 0 new warnings; leakcheck -mode fail 0 hits
result: "Docs only, not verified live. Plan classifies 36 unmerged branches: LAND 20, FOLD 6, TAG-AND-DELETE 5, KEEP 5; 4 more are merged and plain-deletable. Classification is from subjects and ahead/behind, content not diffed (U)."
credit:
  author: owner
  agents: [Claude Code]
  consent: none
merge: 28360da
```

- [x] 1. Inventory of agent/* branches (ahead/behind, merged state)
- [x] 2. docs/research/branch-cleanup-plan.md
- [x] 3. merge-lines.txt and OWNERS entries
