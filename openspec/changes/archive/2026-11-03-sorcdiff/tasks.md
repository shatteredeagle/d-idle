# Tasks

```yaml
change: sorcdiff
status: archived
agent: sorcdiff
branch: agent/sorcdiff
worktree: d2lod-wt/sorcdiff
skip_specs: true
owns:
  - docs/research/sorcdiff.md
  - tools/emu/sorcdiff_*
  - openspec/changes/sorcdiff/**
exclusive:
  - fork-branch:local/sorc-lightning-seq
gates:
  - "local/sorc-lightning-seq is two commits on upstream main 005c7cab, numeric noreply identity, private remote only"
  - "changecheck 0 warnings, leakcheck 0 hits"
result: "docs and emulator tools only"
credit:
  component: Lightning and Chain Lightning cast length (real-Fight test and fix), Sorceress skill use table
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 013512a
```

## 1. Work

- [x] 1.1 skills the lightning build uses, from the table, the combat profile, the leveler and the gate minute archives
- [x] 1.2 cross with the ledger, the traces and the upstream state of 005c7cab
- [x] 1.3 build worker differentials: cast length through the real Fight (base 802 of 3609 differ, head 0)
- [x] 1.4 clean re-runs on 005c7cab: skill_damage, static_field, chill_length, next_hit, armor_hit, freeze_rules 0 differ (missile_hit not re-run)
- [x] 1.5 fix branch local/sorc-lightning-seq pushed to the private remote
- [x] 1.6 docs/research/sorcdiff.md, merge-lines.txt, OWNERS lines
