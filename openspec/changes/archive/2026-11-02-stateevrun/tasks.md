# Tasks

```yaml
change: stateevrun
status: archived
agent: stateevrun
branch: agent/stateevrun
worktree: ~/code/d2lod-wt/stateevrun
skip_specs: true
owns:
  - docs/research/stateevrun-trace.md
  - tools/emu/ser_events.py
  - tools/emu/ser_run.sh
  - openspec/changes/stateevrun/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs only; see the doc for tags and limits"
live: "docs and emulator scripts only, nothing run live"
merge: d202735
```

- [x] harness ser_events.py (event 11 vs resistance, swing resolution, whole chain)
- [x] counted runs on the worker host
- [x] trace doc stateevrun-trace.md
- [x] merge-lines.txt, OWNERS, checks
