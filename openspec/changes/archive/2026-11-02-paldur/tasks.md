# Tasks

```yaml
change: paldur
status: archived
agent: paldur
branch: agent/paldur
worktree: d2lod-wt/paldur
skip_specs: true
owns:
  - docs/research/paldur-trace.md
  - tools/emu/pal_wear.py
  - tools/emu/pal_wear_test.cpp
  - tools/emu/pal_build.sh
  - tools/emu/pal_run.sh
  - tools/emu/pal_callgraph.py
  - tools/emu/pal_tables.py
  - openspec/changes/paldur/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs only; first or second pass, not yet independently verified"
merge: b415aef
credit:
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifypaldur, independent run)"]
  consent: none
```

- [x] trace the call chain (FUN_0057d4f0 -> FUN_0057d3d0 -> FUN_00559e30)
- [x] differential (armor 0 differ, weapon 0 differ except the javelin gate)
- [x] block / dodge flag run (FUN_0057ec10)
- [x] call-graph scan (missile paths)
- [x] write the trace doc, merge lines
