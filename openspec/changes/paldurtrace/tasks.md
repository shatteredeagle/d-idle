# Tasks

```yaml
change: paldurtrace
status: in-progress
agent: chuckado
branch: paldur-trace
skip_specs: true
owns:
  - docs/research/durability-trace.md
  - tools/emu/verify_paldur.py
  - openspec/changes/paldurtrace/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "Offline Unicorn harness run (100,128 cases with seed 0x50414c44, 0 mismatches). Verified weapon wear (4% melee, 10% throwable), armor wear (10% on chosen armor slot, 3:5:4:4:2:2:2 weights), and zero wear on block/miss/dodge."
```

## 1. Work

- [x] 1.1 Trace the call chain (FUN_0057d4f0 -> FUN_0057d3d0 -> FUN_00559e30)
- [x] 1.2 Verify block / dodge handling in FUN_0057ec10
- [x] 1.3 Implement Unicorn harness tools/emu/verify_paldur.py (100,128 cases, seed 0x50414c44)
- [x] 1.4 Write trace doc docs/research/durability-trace.md
