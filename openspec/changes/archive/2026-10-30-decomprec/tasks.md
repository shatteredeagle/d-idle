# Tasks

```yaml
change: decomprec
status: archived
merge: 16cf3e5
agent: DECOMPREC
branch: agent/decomprec
worktree: ~/code/d2lod-wt/decomprec
owns:
  - docs/research/movement-and-recovery-trace.md
  - docs/research/upstream-pr-candidates-movement.md
  - tools/emu/move_*.py
exclusive: []
gates:
  - "every address cited is read in game.exe 1.14d or run in the oracle (V) or tagged S/U"
  - "nothing pushed, posted or filed on GitHub; no game instance started"
result: "Walk/run: engine speed identical (velocity/16 subtiles a tick, oracle V vs d2d cells_per_sec); the d2d bot only walks (never sends cmd::Run), the original runs while stamina lasts: ratio 1.0..1.5, fix on our side. Hit recovery: game.exe GH rate = speed x (50 + E)/100 (oracle V): Sorc 15 vs d2d 8 ticks, Fallen 13 vs 7, Zombie 16 vs 8; decision by hit class with 50/75 % bands vs d2d 1/8 and 1/12. NO PR drafted (user rule 2026-10-30: proof incomplete, server-side monster mode end U). Not verified live."
```

## 1. Work

- [x] 1.1 walk/run speed: rate branch of `00623f50`, velocity base `00621360`, path step `0064fef6`, stamina
- [x] 1.2 hit recovery: `0057cb00` (hit size by hit class), get-hit rate branch, GH length per monster
- [x] 1.3 d2d comparison (moving_anim_rate, hurt, set_mode, stamina_frame) and the oracle
- [x] 1.4 PR candidates recorded as findings, PROOF: incomplete (no branch content: user rule of 2026-10-30)
