# Tasks

```yaml
change: decompwake
status: archived
agent: decompwake
branch: agent/decompwake
exclusive: []
gates:
  - "every rule in the trace doc is tagged V (game.exe read or run under unicorn), V-up, V-d2d, D, S or U with an address or file:line"
  - "the differential runs from upstream's tools/emu and ends with ok (0 differ on monster modes 0..0xf and on the unstunned schedule)"
merge: ea13c89
result: "clean negative: game.exe's wake chain (target search under aidist 0x23 = 35, no row sets aidist; no-target cadence 10 / nearest - 10 / 25 or wander 5 when hit or on a door word; scheduler aidel 0 -> 15, +0x2d stunned, skip when an event is pending later; the rally copies a command only) equals d2d's; oracle 85,820 + 55,268 cases, 0 differ; verifier verdict in the doc. Open U: which rooms think, the first think after spawn, a think due during GH. Not verified live; nothing pushed or posted."
```

## 1. Work

- [x] 1.1 Read the riskmap entry, monster-attack-trace, spawn-density-trace, d2d-gaps, d2d-fidelity section 3, the PR standard and the proof docs
- [x] 1.2 game.exe: FUN_005b1740, FUN_005b1650, FUN_005de890, FUN_005de080, FUN_005a73e0, FUN_005415a0, FUN_005a8030, FUN_0058f730, FUN_005734e0, FUN_005dd2b0 (disasm.py, V); MonStats aidist / aidel / aip (D)
- [x] 1.3 d2d: search_target, think()'s no-target branch, idle / set_off / set_mode, the Fallen rally, near_players, make_monster (V-d2d, file:line)
- [x] 1.4 Oracle `tools/emu/wake_cadence.py`: stand 85,820 / 0, aidel 55,268 / 0; broken model 704 (the check bites)
- [x] 1.5 Independent adversarial verifier (Fable, read-only), verdict quoted in the doc
- [x] 1.6 Trace doc <= 120 lines, card, OWNERS, changecheck 0 warnings, leakcheck fail 0, openspec validate --all --strict; cleanup on the worker host (agentreap list empty)
