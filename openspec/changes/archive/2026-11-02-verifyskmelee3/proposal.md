## Why

The candidate fork branch for the SKMELEE3 upstream PR (melee blow keeps its 256ths, undead/Blunt bonus) needs an independent verification with an own harness before the PR hold is lifted.

## What Changes

- `docs/research/pr-skmelee3-verifier.md`: verdict READY / READY WITH CHANGES / NOT READY, own base-vs-head counts, reader audit, det scenarios, CI-parity results, claim-by-claim check of the author's draft.
- Own harness under `docs/research/pr-skmelee3-verifier/` (scripts only).
- No Go code, no game, no instance; nothing pushed or posted.
