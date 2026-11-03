# Tasks

```yaml
change: tierbdesign
status: archived
agent: tierbdesign
branch: agent/tierbdesign
worktree: d2lod-wt/tierbdesign
skip_specs: true
owns:
  - docs/research/tierb-design-20261102.md
  - openspec/changes/tierbdesign/**
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs only"
credit:
  component: TIERBDESIGN
  author: owner
  agents: [Claude Code]
  consent: none
merge: 780d857
```

- [x] 1 Time-source inventory with evidence tags
- [x] 2 Three stepping designs (cost, risk, EULA, A:B question)
- [x] 3 Statistical alternative and sample sizes; recommendation
- [x] 4 Staged card list and fidelity-gate mapping
- [x] 5 User decisions
- [x] 6 kitb against kitk2 pair reading
- [x] 7 changecheck and leakcheck
