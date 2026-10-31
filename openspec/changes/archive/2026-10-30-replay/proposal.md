# replay: determinism + replay of d2d runs

## Why

The lockstep fork is deterministic for a given seed and input sequence (docs/d2d-fork.md section 5, docs/d2d-loop.md phase 0: three runs, identical monster dumps). Nothing yet captures the inputs of a run, so a soak, a death or a bug can only be described, never replayed. The user's idea (2026-10-30 dialog): "if we can play back a whole run from the seed and inputs that would make a killer replay feature". A replay file is also the determinism oracle: play it back on another build and the first divergent tick says whether the build or the bot changed the game.

## What Changes

- `internal/d2dreplay`: versioned, gzip-compressed replay format (header, ordered control verbs with tick numbers, state-hash checkpoints, trailer), writer, reader, the passive tap (a one-client unix-socket proxy in front of d2d that records every mutating verb and its step count), the player (lockstep re-run with checkpoint hash verification, first divergence reported), a d2d launcher from the header recipe.
- `cmd/d2dreplay`: `record` (tap: listen for the bot or script, forward to d2d, write the file), `play FILE` (start a hidden d2d under the lab lock from the same recipe and seed, replay, print OK or the first divergence), `info FILE`, `render FILE --out DIR` (screenshot verb at a virtual-time stride, optional ffmpeg video, speed factor).
- `docs/d2d-replay.md` (format, commands, limits, results) and a small demo replay under `docs/probes/replay/`.
- ADDED capability spec `replay`.
- Lab lock: `play`/`render` and the live proofs take `/tmp/d2d-lab.lock` with `flock -w 3000` only while a d2d runs (queue behind other holders); no exclusive claim on the card because d2d-lan-oracle-demo holds one.
