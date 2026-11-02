# Tasks

```yaml
change: verifyskresist
status: archived
agent: verifyskresist
branch: agent/verifyskresist
worktree: ~/code/d2lod-wt/verifyskresist
skip_specs: true
owns:
  - openspec/changes/verifyskresist/**
  - docs/research/verify-skresist.md
  - tools/emu/vr_dis.py
  - tools/emu/vr_tbl.py
  - tools/emu/vr_resist.py
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
merge: 0175d16
```

## 1. Work

- [x] 1.1 Own look at the four functions and own models
- [x] 1.2 Differential on two seeds
- [x] 1.3 Doc docs/research/verify-skresist.md
