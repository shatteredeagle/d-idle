## Why

Gate 2 asks the run to "hire the merc". The hirereader design (docs/research/hirereader-design.md, branch agent/hirereader) found bot-only gaps that keep hired, alive-and-following and revived mercs invisible: B1 (only the Rogue code mapped), B2 (a dead merc whose corpse is gone read as not owned), B4 (revive quote unread).

## What Changes

- B2: `present:false,dead:true` now gives an owned, dead merc (revivable) instead of "no merc".
- B1: codes GU, IW, 0A map to classes 338, 359, 561.
- B4: the Reviver parses the "Resurrect <name>: <cost>" line (`ParseQuote`), refuses to click when the purse is below it, exposes `Quote()`, and the `merc: revived` log carries the quote as `estimate`.
- Judge (ops repo, separate commit): `HIRE_RE` also counts the bot's real `merc: hired class=` line. Notes in docs/research/hirefix.md.
- Not done: B3 (a Hirer over `menu`) is bot-only but not small; the d2d dump keys (private fork) are untouched.

## Impact

internal/d2dsrc/player.go, tables.go, internal/d2dmerc, cmd/d2bot/mercwire.go, docs. No spec delta.
