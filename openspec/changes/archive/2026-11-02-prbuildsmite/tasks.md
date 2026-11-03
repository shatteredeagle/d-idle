# Tasks

```yaml
change: prbuildsmite
status: archived
agent: prbuildsmite
branch: agent/prbuildsmite
worktree: d2lod-wt/prbuildsmite
skip_specs: true
owns:
  - docs/research/pr-smite-block-draft.md
  - tools/emu/sbp_probe.cpp
  - tools/emu/sbp_run.sh
  - tools/emu/sbp_ci.sh
  - openspec/changes/prbuildsmite/**
exclusive:
  - fork-branch:local/smite-block-v1
gates:
  - "suites test_rules test_game_rules test_monsters test_monster_thinks test_monster_ai test_combat test_game green on the head"
  - "counted differential base vs head, at least 20,000 cases against game.exe"
merge: d3ec289
result: "docs and tools; PR candidate private"
live: "offline: no game; game.exe's own functions run under the emulator on the worker host"
credit:
  component: SMITEBLOCK
  author: owner
  traced_by: [chuckado]
  verified_by: [verifysmite]
  agents: [Claude Code]
  consent: {chuckado: asked}
```

- [x] 1 Fork branch local/smite-block-v1 (-final squashed) from upstream main 82ea6db4, numeric noreply identity
- [x] 2 Unit tests (blocked Smite lands; plain swing, enhanced swing, kick stopped; chance 0 vs 100)
- [x] 3 d2d probe and counted differential base vs head against game.exe (two seeds)
- [x] 4 Animation: decided with the disassembly, stated as a limit
- [x] 5 CI parity (g++, clang++, extra flags, pr_guard, presubmit --quick, seven suites)
- [x] 6 Draft doc, card, OWNERS, merge lines, changecheck, leakcheck
