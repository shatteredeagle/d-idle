## Why

Coverage gap "missiles", second pass. The first pass (covmissiles) ran the table values of all 684 rows and the element hit of 14 skills; monster-missile-trace and missilediff covered the MissA rows. The rows a monster fires through a Skills.txt skill (Fallen Shaman fire, Unraveler bolt, Andariel's spray and poison bolt, vampire / Countess fire walls, Sand Leaper) go through d2d's `andariel_missile`, a different code path, and were READ-ONLY or untested.

## What Changes

- `docs/research/covmissiles2-trace.md`: game.exe's `FUN_0059fa30` record of 17 skill-sourced rows against the real d2d loader, `rules::row_damage`, `ai.hpp missile_blow` and `rules::monster_blow`: 27,000 cases (21,926 compared), deviations D1 to D4, bot findings, drill recipe, matrix text.
- `tools/emu/cm2_*`: the oracle (game side, d2d probe, compare, run script).
- `docs/coverage/ledger.tsv` and `rule-coverage.md` (regenerated): rows 22, 24, 77, 130, 346, 653 TESTED-0-DIFF, row 32 TESTED-DEVIATION; the READ-ONLY line of monster-missile-trace gives those rows up.
- No Go code, no d2d change, no PR candidate (the poison finding already belongs to a local branch; no after-run here).
