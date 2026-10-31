# Tasks

```yaml
change: rendergh
status: archived
skip_specs: true
agent: rendergh
branch: agent/rendergh
worktree: ~/code/d2lod-wt/rendergh
owns:
  - docs/research/pr-get-hit-render-*.md
  - tools/emu/hit_render*
  - openspec/changes/rendergh/**
exclusive:
  - upstream-pr:get-hit-render
gates:
  - "differential vs game.exe's client step and draw (FUN_004b1280, FUN_00621810): >= 100000 random + boundary cases, mismatches before > 0, 0 after the patch"
  - "upstream tests green on the patched worktree; pr_guard 0 FAIL 0 WARN"
  - "one independent adversarial verifier report attached"
  - "nothing pushed or posted to GitHub"
merge: ccd653f
result: 'get-hit drawn-frame PR candidate (monsters + merc): differential vs game.exe FUN_004b1280 + FUN_00621810 over 55,089 kind-1 cases / 1,747,580 ticks: 1,488,667 ticks differ before, 0 after; the player path measured (136,651 ticks differ) and left as a limit; patch 57bd4b56 on fix/get-hit-render stacked on eb89646e (base moved to 9006a688: rebase listed); nothing pushed; not verified live'
```

## 1. Trace (game.exe 1.14d, addresses only)

- [x] 1.1 the client's monster tick (FUN_004b13a0 -> FUN_004b1280) and its mode table row for GH (FUN_004af400, 0x7252e0)
- [x] 1.2 the over test, the step, the drawn frame (FUN_006217c0, FUN_00623e00, FUN_00621810); the client's mode start (FUN_00624390)
- [x] 1.3 the player's client tick (FUN_00463390, 0x711e00 row 4): the same step, the mode ended a tick early; left out of the claim

## 2. Differential driver

- [x] 2.1 `tools/emu/hit_render.py` gen / game / compare; `hit_render_harness.cpp` (d2d's old lines)
- [x] 2.2 110,836 cases, kind 1: 1,488,667 of 1,747,580 ticks differ before, 0 after; kind 0 measured (not claimed)

## 3. Patch

- [x] 3.1 `rules::get_hit_frame`, `Unit::get_hit_rate`, `get_hit_elapsed`, the merc's FHR in the view; test_combat asserts and `--get-hit-frames`; combat.md
- [x] 3.2 ninja test_combat test_game; test_combat ok; test_game 23 OK; pr_guard 0 / 0; presubmit --quick (1 FAIL inherited from #136's doc path, answered)

## 4. Verifier and docs

- [x] 4.1 independent adversarial verifier (general-purpose, Fable, read-only)
- [x] 4.2 proof, draft, evidence pack; changecheck, leakcheck, openspec validate
