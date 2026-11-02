# Tasks

```yaml
change: prcandidates
status: archived
agent: prcandidates
branch: agent/prcandidates
worktree: d2lod-wt/prcandidates
skip_specs: true
owns:
  - openspec/changes/prcandidates/**
  - docs/research/pr-candidates-20261102.md
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck -mode fail 0"
result: "docs only"
merge: 4ffc246
```

## 1. Work

- [x] 1.1 List fork branches and compute what is not on upstream main
- [x] 1.2 Read the PR drafts, verifier records, ledger and backlog
- [x] 1.3 Write docs/research/pr-candidates-20261102.md with the ranking
