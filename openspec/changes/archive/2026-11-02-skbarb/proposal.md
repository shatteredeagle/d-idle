## Why

Coverage gap (docs/coverage/rule-coverage.md): the Barbarian rows (skills.bar, 30 of 30) had no game.exe comparison, and the Barbarian plans (War Cry endgame, melee leveling) rest on Bash, Frenzy, Whirlwind, Howl, War Cry and the buffs.

## What Changes

- docs/research/skbarb-trace.md: trace of the Barbarian skills (game.exe vs d2d vs bot), deviations D1..D7, bot findings B1..B5, no PR candidate, drill recipe, matrix cell text.
- tools/emu/skbarb_*: calc and skill-physical differential (27,800 cases, 0 differ), Whirlwind target-search run, Howl record readers, war-cry ring flights.
- docs/coverage/ledger.tsv and rule-coverage.md (regenerated): skills.bar 30 rows TESTED-0-DIFF on the calc and physical aspect.

## Impact

Docs and emulator tooling only. No Go, spec, policy, ADR or glossary change. Glossary candidates for the coordinator: "subtile (one fifth of a cell; the unit of unit-search radii)", "justhit / NextHit window".
