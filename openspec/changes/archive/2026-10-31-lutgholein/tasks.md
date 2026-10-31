# Tasks

```yaml
change: lutgholein
status: archived
agent: lutgholein
branch: agent/lutgholein
worktree: ~/code/d2lod-wt/lutgholein
owns:
  - docs/research/lutgholein-*.md
  - tools/emu/lut_*.py
exclusive:
  - instance:i1
  - port:4000
gates:
  - "cause of the Lut Gholein whisper-town vs drlg-dump mismatch stated as artifact or bug with evidence a reader can re-run"
merge: 6be61ae
result: "Lut Gholein: BOTH: (a) comparison artifact: drlg-dump brings 49 rooms up in list order vs live client order (player room, 3x3 near list, ring); feeding the live order reproduces every room cell-for-cell on 2 seeds; (b) REAL d2d bug (upstream): Lut Gholein room seeds two level-RNG steps behind game.exe (live room k seed == d2d seed of room k+2, 47/47; emulator: game.exe 49/49, d2d 0/49; 20 seeds: 0/20 match, 20/20 equal game.exe shifted by two rooms); PR candidate PROOF incomplete: mechanism untraced (which game.exe function takes the two extra steps before the per-room roll: candidates FUN_00667970, FUN_00666ed0)"
```

## 1. Work

- [x] 1.1 Read LANORACLE's capture and d2d's `drlg-dump` / whisper `town` formats; compare offline
- [x] 1.2 Find the transform or the real mismatch (room list, tiles, DS1 file)
- [x] 1.3 Re-capture Lut Gholein live on i1 (Navi) and verify the finding
- [x] 1.4 Write docs/research/lutgholein-trace.md (V/S/U), PR candidate status
- [x] 1.5 Cleanup: stop i1, nothing of ours left running
