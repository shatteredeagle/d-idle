## Why

The fork candidate pr/player-roll (four commits on upstream main: element roll top, physical roll in 256ths, the -100 physical floor, plus an oracle) is meant for an upstream PR. The author's evidence needs an independent check with a separate harness before anything goes upstream.

## What Changes

- docs/research/pr-playerroll-verifier.md: diff and merge check, own counts for the physical roll, the floor and the Sorceress / Necromancer element rolls, caller audit, det hashes, warnings-as-errors build, suites, overclaims, verdict.
- Docs only; harness stays on the worker host under the lab directory. No Go code, no game.exe, nothing pushed or posted.

## Impact

None on shipped code. Result docs/research/pr-playerroll-verifier.md.
