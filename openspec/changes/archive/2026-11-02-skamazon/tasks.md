# Tasks

```yaml
change: skamazon
status: archived
agent: skamazon
branch: agent/skamazon
worktree: ~/code/d2lod-wt/skamazon
skip_specs: true
owns:
  - docs/research/skamazon-trace.md
  - tools/emu/ama_*
  - openspec/changes/skamazon/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; coverage.py --check; tools/ci/check.sh"
result: "docs only; see the doc for tags and limits"
live: "docs and emulator only, nothing run live"
merge: 36ac3a3
```

- [x] calc differential (ama_calc.py, nec_test.cpp reused)
- [x] elemental missile hit differential (ama_missile_hit.py, sk_missile_hit_test.cpp reused)
- [x] read the Amazon start/do functions (table walk; disassembly only where noted)
- [x] trace doc, ledger rows, rule-coverage regenerated
- [x] merge-lines
