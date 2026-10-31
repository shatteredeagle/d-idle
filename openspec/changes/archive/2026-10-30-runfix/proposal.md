## Why

The d2d bot only walks (the toggle defaults to walking and nothing sends it), while the original character runs while stamina lasts. DECOMPREC (V by emulator oracle) found the engine speeds identical, so running is 1.5x faster travel; walking is 75 % of per-kill time on d2d (KILLGAP).

## What Changes

- internal/d2dinput: RunPolicy (resume at 25 % stamina, stop below 8 %), applied before every move command through the fork's run-toggle key; opt-in field, wired in d2denv, `D2D_RUN=off` restores the old behaviour.
- docs/d2d-run.md: stamina model probe and the before/after measurement.
