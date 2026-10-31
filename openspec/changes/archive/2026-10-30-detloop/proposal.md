## Why

Two full-bot runs on the lockstep d2d fork with the same seed, save and build diverge (docs/d2d-loop.md "Determinism of lab5"),
although scripted walks and fights are deterministic. Replay (seed + start save + build hash + input log, lockstep playback) and
clean A/B fidelity measurements need the bot to send the same command stream on the same world. This change finds every source
of divergence between two same-seed runs, removes the bot-side ones and lists the rest with the exact fork request.

## What Changes

- a divergence finder: `D2DCTL_JOURNAL` records every command the bot sends with its lockstep tick and a reply digest, plus a
  state hash every N ticks; `cmd/d2ddet compare` reports the first differing record and classifies it (bot-side or game-side);
  `tools/d2ddet/pair.sh` runs the pair on the worker host
- bot-side fixes in the d2d packages (and shared packages where unavoidable, flagged Game.exe-affecting)
- docs/d2d-determinism.md: sources, evidence, fixes, remaining blockers, fork requests, checkpoint and replay file format
  for the REPLAY agent (cmd/d2dreplay, internal/d2dreplay: not touched here)
