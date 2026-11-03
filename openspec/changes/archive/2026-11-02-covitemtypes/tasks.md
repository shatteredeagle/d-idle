# Tasks

```yaml
change: covitemtypes
status: archived
agent: covitemtypes
branch: agent/covitemtypes
worktree: ~/code/d2lod-wt/covitemtypes
skip_specs: true
owns:
  - docs/research/covitemtypes-trace.md
  - tools/emu/cit_oracle.py
  - tools/emu/cit_d2d_test.cpp
  - tools/emu/cit_build.sh
  - tools/emu/cit_dis.sh
  - tools/emu/cit_diff.py
  - tools/emu/cit_diff2.py
  - tools/emu/cit_diff3.py
  - tools/emu/cit_diff4.py
  - docs/coverage/ledger.tsv
  - docs/coverage/gaps.tsv
  - docs/coverage/rule-coverage.md
  - openspec/changes/covitemtypes/**
exclusive: []
gates:
  - "every ledger case count appears in the trace; python3 tools/coverage/coverage.py --check clean"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs/tools only; see the doc for tags and limits"
merge: 62e78ff
```

## 1. Work

- [x] 1.1 Trace the type walk, closure matrix, getters, slot, hands and socket functions in game.exe
- [x] 1.2 Emulator oracle and d2d probe (real tables), counts
- [x] 1.3 Bot comparison (file:line), latent deviations, drill recipe
- [x] 1.4 Trace doc, ledger line, gap row, rule-coverage.md regenerated
- [x] 1.5 merge-lines.txt

## 2. Open

- [ ] 2.1 Independent verifier (only if a later change turns B1 or B3 into a fix)
- [ ] 2.2 Live drills of the recipe (needs a lab slot)
