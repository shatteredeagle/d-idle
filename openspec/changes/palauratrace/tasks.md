# Tasks

```yaml
change: palauratrace
status: in-progress
agent: chuckado
branch: palaura-trace
skip_specs: true
owns:
  - docs/research/pal-aura-trace.md
  - tools/emu/verify_palaura.py
  - openspec/changes/palauratrace/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "Offline Unicorn harness run (300,000 cases across 3 suites, 0 mismatches; V read + self-check, offline, live unverified). Verified 19-bit target filter FUN_0056b3e0, area search FUN_0056b7e0, pulse damage FUN_0056e0c0, Prayer mana payment & state 85 gate at 0x5cf239 in FUN_005cf010, and player mana regeneration in FUN_005806f0."
```

## 1. Work

- [x] 1.1 Trace target filter FUN_0056b3e0 and area search FUN_0056b7e0 in game.exe 1.14d
- [x] 1.2 Trace pulse damage FUN_0056e0c0 for offensive pulse auras
- [x] 1.3 Trace Prayer mana payment & state 85 gate in FUN_005cf010 and FUN_005806f0
- [x] 1.4 Implement Unicorn harness tools/emu/verify_palaura.py (300,000 cases)
- [x] 1.5 Write research trace doc docs/research/pal-aura-trace.md
