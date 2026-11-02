# Tasks

```yaml
change: skgearar
status: archived
agent: skgearar
branch: agent/skgearar
worktree: ~/code/d2lod-wt/skgearar
skip_specs: true
owns:
  - openspec/changes/skgearar/**
  - docs/research/sk-gearar-trace.md
  - tools/emu/sk_gearar.py
  - tools/emu/sk_gearar_dis.py
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
merge: b0637e1
```

## 1. Work

- [x] 1.1 Read FUN_0057d8a0 and its call in FUN_0057d9b0
- [x] 1.2 Oracle tools/emu/sk_gearar.py, 100,960 cases
- [x] 1.3 Trace docs/research/sk-gearar-trace.md
- [x] 1.4 Ledger row, card, OWNERS
- [ ] 1.5 Stats 179 / 120 run and a d2d-side test (follow-up)
