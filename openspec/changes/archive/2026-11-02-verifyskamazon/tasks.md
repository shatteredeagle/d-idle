# Tasks

```yaml
change: verifyskamazon
status: archived
agent: verifyskamazon
branch: agent/verifyskamazon
worktree: ~/code/d2lod-wt/verifyskamazon
skip_specs: true
owns:
  - docs/research/verifyskamazon-trace.md
  - tools/emu/vam_*
  - openspec/changes/verifyskamazon/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; coverage.py --check; tools/ci/check.sh"
result: "docs only; see the doc for tags and limits"
live: "docs and emulator only, nothing run live"
merge: 0489b0f
```

- [x] read FUN_0059f940, collision tail, FUN_005a8910, FUN_0045c3e0
- [x] own calc, pierce and roll harnesses run on the worker host
- [x] verdicts per claim and per ledger line
- [x] trace doc, merge-lines
