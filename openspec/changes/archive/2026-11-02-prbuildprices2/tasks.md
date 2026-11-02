# Tasks

```yaml
change: prbuildprices2
status: archived
agent: prbuildprices2
branch: agent/prbuildprices2
worktree: d2lod-wt/prbuildprices2
skip_specs: true
owns:
  - openspec/changes/prbuildprices2/**
  - docs/research/pr-vendor-repair-gate-draft.md
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs only; PR hold in force"
merge: 08f85a5
credit:
  component: Vendor repair gate
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifyprices, independent run)"]
  consent: none
```

## 1. Work

- [x] 1.1 Branch local/vendor-repair-gate-v3 from upstream main, da80d0ab alone, text corrected
- [x] 1.2 Suites, base vs head counts, CI parity, throwaway merge with #174
- [x] 1.3 Draft docs/research/pr-vendor-repair-gate-draft.md
