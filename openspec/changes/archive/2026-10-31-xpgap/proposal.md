## Why

The d2d bot reaches about 4985 XP/h against the early baseline of 6600 (docs/d2d-soak-full-fix.md). The engine matches Game.exe, so the gap is bot behaviour: 54 % of the time is "Recovery: farming gold" and 28 % town chores. Diagnose from the killtime and minute lines, fix the top causes behind knobs (default off), measure before/after on one binary.

## What Changes

- knob `resource.recoveryXPEarly`: a recovering character enters the better-XP area earlier
- knob `resource.povertySellMin`: poor characters make fewer, bigger sell trips
- `docs/d2d-xp-gap.md`: diagnosis, causes, fixes, measured A/B, ranked next suspects
