## Why

Four merged fixes (fixstall, fixmisc, fixrun, fix-portal2) were proven offline only. DRILLS6 runs them live on the hidden slot and records the numbers, including whether the bot walks instead of running after a fresh game.

## What Changes

- Drills D-sync and D-run (group F), D6.d accepts ErrLevelChanged after the pocket escape (internal/drill, cmd/d2drill only).
- Results in docs/probes/drills (drills6 files, README section DRILLS6), primitives rows, each fix card gets a `live:` line.
- No product code changes: failures are findings (F7, F8, F9 in the README section).
