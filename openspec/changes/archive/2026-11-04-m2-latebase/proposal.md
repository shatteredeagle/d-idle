## Why

The early baseline holds only fresh-character (clvl 1 to 7) Game.exe runs, so a d2d run in the clvl 8 to 15 band has nothing to be compared with, and Game.exe is frozen (no new original runs). The user decided on 2026-10-31 (D5, docs/roadmap.md): build the late baseline for XP per hour and kills per hour from a cheat-levelled clvl 12 throwaway character carrying the M1 kit, on hidden d2d, every cheat recorded in the logged ledger. The d2-sme (2026-10-31): fair for XP/h and kills/h with XP set to the table threshold, points spent by the leveler, the baseline kit; NOT valid for deaths or gold.

## What Changes

- tools/d2dsoak/baselines-late.json: a baseline of runs from a clvl 12 start (XP/h and kills/h only; deaths and gold are not gated against it), each with a source line and the cheat ledger entry ids.
- docs/d2d-baselines-late.md: protocol (character, M1 kit, cheat reasons, seeds, virtual hours), what is V and what is U, why deaths and gold are excluded.
- Cheats only through `internal/cheat` / `cmd/d2cheat` (docs/cheats.md), each with a reason, on a throwaway character, offline, recorded in the ledger; no cheat during the measured window.
- Delta: fidelity-gate ADDED `Late baseline`.

## Dependencies

After m1-liveproof (the M1 kit and town basics must be the ones measured) and m2-abdesign (paired protocol). Holds one d2d slot while running.
