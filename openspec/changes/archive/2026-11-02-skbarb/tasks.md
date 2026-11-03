# Tasks

```yaml
change: skbarb
status: archived
agent: skbarb
branch: agent/skbarb
worktree: ~/code/d2lod-wt/skbarb
skip_specs: true
owns:
  - docs/research/skbarb-trace.md
  - tools/emu/skbarb_*
  - openspec/changes/skbarb/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; coverage.py --check; tools/ci/check.sh"
result: "docs only; see the doc for tags and limits"
live: "docs and emulator only, nothing run live"
merge: 0f4feb7
```

- [x] calc and skill-physical differential (skbarb_calc.py, skbarb_test.cpp, skbarb_run.sh)
- [x] Whirlwind target search, Howl readers, war-cry ring flights (skbarb_ww.py, skbarb_howl.py, skbarb_ring.py)
- [x] read the whirl timer, Frenzy, Howl hit, do 68
- [x] trace doc, ledger, rule-coverage regenerated
- [x] merge-lines
