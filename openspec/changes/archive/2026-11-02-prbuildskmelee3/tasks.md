# Tasks

```yaml
change: prbuildskmelee3
status: archived
agent: prbuildskmelee3
branch: agent/prbuildskmelee3
worktree: d2lod-wt/prbuildskmelee3
skip_specs: true
owns:
  - docs/research/pr-skmelee3-draft.md
  - tools/emu/sk_melee2.py
  - tools/emu/sk_melee2_test.cpp
  - tools/emu/sm3_probe.py
  - tools/emu/sm3_run.sh
  - openspec/changes/prbuildskmelee3/**
exclusive:
  - fork-branch:local/skmelee3-v1
gates:
  - "suites test_rules test_game_rules test_monsters test_monster_thinks test_monster_ai test_combat test_game green on the head"
  - "counted differential base vs head over at least 100,000 cases"
result: "docs only; PR hold in force"
live: "offline: no game; game.exe's own functions run under the emulator on the worker host"
merge: 7620862
credit:
  component: Melee fine physical (D1) and undead / Blunt bonus (D2), first build
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifyskmelee3, independent run)"]
  consent: none
```

- [x] 1 Fork branch local/skmelee3-v1 from upstream main: D1 and D2 commits with tests
- [x] 2 Differential base vs head over >= 100,000 cases per claim
- [ ] 3 Det scenarios on lab builds of base and head: not run (checklist in the draft)
- [x] 4 CI parity (gcc and clang flags, include audit, MSVC read, pr_guard, presubmit --quick)
- [x] 5 Draft docs/research/pr-skmelee3-draft.md
- [ ] 6 changecheck and leakcheck
