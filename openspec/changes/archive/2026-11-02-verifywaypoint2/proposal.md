## Why

Board item #8 (VERIFY-WAYPOINT, second pass): an independent re-check of the claim that game.exe refuses a waypoint use only within 10 s of the player's last hostile declaration (pcdata+0x160), while d2d counts the window from every arrival. The first verifier reused the author's oracle and read d2d only; this pass runs d2d's own binary too.

## What Changes

- New doc docs/research/verifywaypoint2-trace.md (HOLDS / LIMITS / FAILS, reproduction commands, deviations D1.., bot findings B1..).
- New emulator-side scripts tools/emu/vw2_*.py and vw2_d2d.sh (worker host only). No Go changes, no game, no ledger row (waypoint is not a coverage subsystem).

## Impact

Docs and tools/emu only. No spec delta.
