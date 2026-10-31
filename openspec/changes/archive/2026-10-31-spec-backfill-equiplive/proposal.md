## Why

Card equiplive (archived 2026-10-30) changed decisions and log lines in internal/town (equip click logging, skipping an item the game refused) but merged with skip_specs; docs/spec-skip-audit.md judged it needs-spec (effort S). It sits next to the existing no-armor-rebuy-loop requirement in early-game-policy, which it does not restate.

## What Changes

- Delta ADDED to early-game-policy: logged slot clicks, refused items skipped until the character changes, a failed equip returns the item and says why
- Nothing in internal/** or cmd/** changes
