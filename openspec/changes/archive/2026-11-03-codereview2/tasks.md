# Tasks

```yaml
change: codereview2
status: archived
agent: codereview2
branch: agent/codereview2
worktree: d2lod-wt/codereview2
skip_specs: true
owns:
  - docs/research/codereview2.md
  - openspec/changes/codereview2/**
exclusive: []
gates:
  - "changecheck -cli 0 new warnings; leakcheck -mode fail 0; merge_lines_check --changed"
result: "review doc written offline (code read, levels.txt read on the worker host, the new areareach tests run); not verified live"
credit:
  author: owner
  agents: [Claude Code]
merge: 4102351
```

- [x] 1 read the diffs a45b9af4..origin/main in internal and cmd, and the four docs
- [x] 2 multi-hop walk rule: limits, loops, danger (R1 to R4)
- [x] 3 XP rank: ties, missing levels, difficulties, gates (R5 to R7)
- [x] 4 CODEFIX and SPECDELTA side effects (R8, R9), test coverage
- [ ] 5 live check (Gate 1 re-run lines in the doc)
