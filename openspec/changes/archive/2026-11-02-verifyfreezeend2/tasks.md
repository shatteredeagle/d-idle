# Tasks

```yaml
change: verifyfreezeend2
status: archived
agent: verifyfreezeend2
branch: agent/verifyfreezeend2
worktree: ~/code/d2lod-wt/verifyfreezeend2
skip_specs: true
owns:
  - openspec/changes/verifyfreezeend2/**
  - docs/research/verifyfreezeend2-trace.md
  - tools/emu/vfe2_run.py
  - tools/emu/vfe2_cmp.py
  - tools/emu/vfe2_tab.py
  - tools/emu/vfe2_all.sh
  - tools/emu/vfe2_build.sh
  - tools/emu/test_vfe2_gh.cpp
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs only; first or second pass, not yet independently verified"
merge: b123648
```

## 1. Work

- [x] 1.1 Read the first pass docs, trace the get-hit start and the driver
- [x] 1.2 Own harness, game side (gh, frz) run on the worker host
- [x] 1.3 d2d probe built and run, compared per case
- [x] 1.4 Trace doc, card, OWNERS, merge lines
