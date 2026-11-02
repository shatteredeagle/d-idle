## Why

The candidate local/nmlevel (private fork, head 9672fc6c on upstream main 6935cf64) claims that game.exe starts a
Nightmare / Hell monster at the area's level (Levels.txt MonLvl2Ex / MonLvl3Ex) unless its MonStats row is noRatio or
boss, and that d2d's base starts it at the MonStats Level(N) / Level(H). An independent check with its own harness is
needed before the PR is opened upstream.

## What Changes

- `docs/research/pr-nmlevel-verifier.md`: verdict (READY / READY WITH CHANGES / NOT READY) with own counts (two new
  seeds, every non-town level, three difficulties), the caller and reader audit, det scenarios on base and head, the CI
  parity checks (g++ / clang++ warnings, clang-tidy "Names and includes"), the suites, and every claim of the author's
  draft against the counts.
- Own harness under `docs/research/pr-nmlevel-verifier/` (game.exe's FUN_00573cb0 in the emulator on real room
  populations; d2d's real make_monster / make_boss in a probe built against base and head).
- No Go code, no game, nothing pushed or posted.
