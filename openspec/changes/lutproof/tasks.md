# Tasks

```yaml
change: lutproof
status: in-progress
agent: lutproof
branch: agent/lutproof
worktree: ~/code/d2lod-wt/lutproof
owns:
  - docs/research/lutgholein-proof.md
  - docs/research/pr-lut-gholein-*.md
  - tools/emu/lut_*.py
exclusive:
  - instance:i1
gates:
  - "the two extra level-seed steps located in game.exe by address (emulator hook log), and the d2d code that skips them named"
  - "differential drlg.py vs drlg-dump, level 40, >= 20 seeds: mismatches before > 0, after 0; other towns and the act 1 Scan/Pops levels unchanged or improved"
  - "independent adversarial verifier report attached; PROOF complete or incomplete stated with the missing piece"
merge:
result: "Traced: game.exe steps Lut Gholein's level seed twice in FUN_00667620 (write 0x667744, the %3 roll to stay for MonStats 204/205 = DS1 monsters 25/26 via monpreset.bin) through FUN_00667970 before the 49 room allocations (FUN_0066b3e0). NOT a d2d generator bug: tools/drlg-dump never opened the folder's patch_d2.mpq (only $D2_PATCH_INSTALLER), so monpreset.bin was missing, no id mapped, no roll, two-room seed lag. Fix on the upstream clone branch fix/lut-gholein-preset-rng (632d5cc2, author the user, tools/drlg-dump only): open the MPQs as open_game_data does. diff_drlg.py level 40 seeds 1-20 tiles/units/seeds 0/20 -> 20/20; levels 1,2,75,103 tiles 10/10 both, units 0,6,0,0 -> 10/10; 109 tiles+units 0/10 -> 10/10; pr_guard 0/0; tests ok. Verifier: HOLDS WITH LIMITS (no-op for a setup with the installer). PROOF complete for the narrowed claim. Nothing pushed; no instance started. Verified in the emulator, not live."
```

## 1. Work

- [x] 1.1 Trace every level-seed write for level 40 in the emulator (lut_trace.py); name the two extra steps
- [x] 1.2 Find why d2d skips them; fix on a fresh branch off upstream/main in the clone; pr_guard, presubmit, tests
- [x] 1.3 Differential before/after with upstream's diff_drlg.py: level 40 seeds/tiles/units 1-20; levels 1, 2, 75, 103, 109 1-10
- [x] 1.4 Independent adversarial verifier (Agent, general-purpose, read-only)
- [x] 1.5 Docs: lutgholein-proof.md, pr-lut-gholein-draft.md, pr-lut-gholein-evidence-pack.md
- [x] 1.6 Cleanup: no instance started, no pids left; report
