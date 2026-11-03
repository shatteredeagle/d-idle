# sorcdiff

## Why

The gate character is the lightning Sorceress (plan table `fallback-lightning-v2`; projreview finding 3), not a Barbarian. Rule coverage says 15 Sorceress skills deviate and 15 are read-only, but nobody had crossed that with the skills the bot actually spends points on and casts. This card makes that table, ranks what is unproven, and runs counted differentials against the real d2d build (upstream main `005c7cab`) on the build worker WSL lane.

## What Changes

- `docs/research/sorcdiff.md`: the table (skill, bot use, cast data, status, PR state), the ranking, the counted differentials, the ready-for-Opus list.
- `tools/emu/sorcdiff_worker_run.sh`: the WSL script that built and ran the differentials (copy of the ops-repo script, so the run can be repeated).
- Private fork (not in this repo): branch `local/sorc-lightning-seq` on upstream main `005c7cab`, two commits (harness, fix), the private remote only.

## Impact

Docs and a script. Nothing pushed to GitHub, no PR, no comment.

Overlap (declared): reads the traces and ledger rows of sorcrest, sorcrest2, skmiss, sklight, sknova, skarmor, skpass, skground2 and the barbdiff idea (a mistaken premise); edits none of their files.
