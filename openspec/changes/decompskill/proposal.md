## Why

Riskmap gap rank 4 (`skill_damage`, docs/research/d2d-fidelity-riskmap.md section 5): d2d's skill damage was never compared with game.exe 1.14d on the unit path (synergies, masteries, the stat the missile carries), only the unit-less tooltip and the roll. Casts per kill of the Sorceress's attack skills depend on it. Trace it in game.exe (offline, the unicorn oracle), prove every deviation with a differential against game.exe's own code, and prepare a PR candidate under docs/process/upstream-pr-standard.md, or a clean negative.

## What Changes

- docs/research/skill-damage-trace.md: the formula with addresses, d2d's version with file:line, the deviations table with the differential results, what was verified clean
- tools/emu/skill_damage.py: the oracle (game.exe's FUN_00644d50 / 00644e40 / 00644f20 with synergies and masteries)
- docs/research/pr-skill-damage-{proof,draft,evidence-pack}.md: the PR candidate under our standard (nothing pushed or posted)
- the local upstream branch `fix/skill-damage` in `~/d2d-lab/pr/skilldmg` on the worker host (3 commits, author the user)
