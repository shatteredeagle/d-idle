# replayfix: harden d2dreplay and the d2dview mux

## Why

docs/research/code-review-2026-10-30.md (reviewdoc faf890b) sections 3 and 8 found: the tap cannot be stopped while a client is attached and never writes its trailer, the replay reader panics or allocates without bound on crafted files, hostile verbs and header args reach a no-auth d2d, the launcher ignores the slot pool, and d2dview watch speed 0.001 freezes the bot.

## What Changes

- tap serves in a goroutine, stops on signal/-max/-idle, closes the client, always writes the trailer, real "err busy", socket safety.
- strict reader: size/entry/line/step/tick caps, hash and verb validation, bad lines fatal unless the cut tail; verb allow-list; `Build.Args` allow-list; recipe validation.
- launcher: slot pool (same lock files as tools/d2dslot.sh), validate before queueing, release after d2d exits; `render -out` checks.
- d2dview: watch speed clamp, pacing outside the lock with a cap, normalised viewer lines, verb-variant tests.
