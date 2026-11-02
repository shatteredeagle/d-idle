# Tasks

```yaml
change: verifyfreeze
status: archived
agent: verifyfreeze
branch: agent/verifyfreeze
worktree: ~/code/d2lod-wt/verifyfreeze
skip_specs: true
merge: c0ff783
owns:
  - openspec/changes/verifyfreeze/**
  - tools/emu/verify_freeze.py
  - tools/emu/verify_freeze_spawn.py
  - tools/emu/verify_freeze_mode.inc
  - docs/research/pr-freeze-proof.md
  - docs/research/pr-bundle-queue.md
  - docs/BACKLOG.md
exclusive: []
gates:
  - "own differential on real d2d builds: per claim mismatches before > 0, after 0, every residual classified; claim isolation per commit"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict"
result: "offline verification (no live game, no d2d slot, nothing pushed): fix/freeze-rules C5 and C3 HOLD WITH LIMITS, C4 HOLDS; own run 146,591 cases through the real Fight::strike, 124,662 differ on ddc8281d, 6 on bbf506c6 (length 0x100001, MulDiv), per claim C5 27,309 / C3 74,031 / C4 23,306 -> 0, each commit clears its own slice; native spawn run settles champions (0x0d, flag 8, chilled) vs minions (0x10, frozen); CI-class checks clean; NOT READY: commit bodies over three lines, det state hashes, Frozen Armor in the body's limits"
credit:
  component: Verifier: freeze rules
  author: owner
  agents: [Claude Code]
  consent: none
```

- [x] 1.1 Re-derive FUN_0057b230, FUN_0057af80, the +0x34 resist row, FUN_00639db0 and the spawn-path flags from the disassembly (before reading the author's docs)
- [x] 1.2 Native spawn run: flag word and FUN_0057b230 outcome per spawned monster
- [x] 1.3 Own generator and harness; real d2d builds before, each claim commit, head; residuals classified
- [x] 1.4 Sweep game.exe and d2d for other freeze paths
- [x] 1.5 CI-class checks, PR size, fast-forward and rebase simulation, commit format
- [x] 1.6 Verifier section, queue row, BACKLOG line, OWNERS
- [x] 1.7 Remove the throwaway worktrees on the worker host; agentreap empty
