# Tasks

```yaml
change: palskilltrace
status: in-progress
agent: chuckado
branch: palskill-trace
skip_specs: true
owns:
  - docs/research/pal-skill-trace.md
  - docs/research/pal-smite-zeal-sacrifice-trace.md
  - tools/emu/verify_palskills.py
  - openspec/changes/palskilltrace/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "Offline formula arithmetic self-check (I, 0 mismatches). Confirmed d2d skill_built implements Smite, Zeal, and Sacrifice; identified deviations D8 (Smite blockable in d2d, V read, S) and D9 (Zeal missing rapid frame rollback in d2d, V read, S, U timing)."
```

## 1. Work

- [x] 1.1 Trace Smite, Zeal, and Sacrifice dispatch in game.exe 1.14d
- [x] 1.2 Verify d2d skill_built implementation in components/game/fight.hpp
- [x] 1.3 Implement formula self-check tools/emu/verify_palskills.py
- [x] 1.4 Write dedicated trace doc docs/research/pal-smite-zeal-sacrifice-trace.md
- [x] 1.5 Update cross-reference link in docs/research/pal-skill-trace.md
