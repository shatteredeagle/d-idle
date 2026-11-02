# Tasks

```yaml
change: alignment2
status: archived
agent: alignment2
branch: agent/alignment2
worktree: ~/code/d2lod-wt/alignment2
skip_specs: true
owns:
  - docs/research/alignment2-trace.md
  - tools/emu/al2_run.py
  - tools/emu/al2_ctx.py
  - tools/emu/al2_peek.py
  - openspec/changes/alignment2/**
exclusive: []
gates:
  - "game.exe native runs on the worker host: counts in the trace (734 rows, 18, 168, 288 cases; 0 differ)"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs only; see the doc for tags and limits"
merge: 0ed9efc
credit:
  component: Undead / Blunt bonus (D2 trace)
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifyskmelee3, independent run)"]
  consent: none
```

## 1. Work

- [x] 1.1 Read FUN_006259b0, its setter and callers in game.exe
- [x] 1.2 Native runs (Align byte, setter/getter, getter shapes, gate)
- [x] 1.3 Trace doc with table, rule, bot findings, drill recipe
- [x] 1.4 merge-lines.txt

## 2. Open

- [ ] 2.1 Live read of stat 0xac on a hired merc and a Shadow (drill in the trace)
