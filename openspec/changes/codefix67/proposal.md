## Why

docs/research/codefix.md section 3 lists codereview1 findings F6 (stash-gold withdraw only when the purse cannot pay the kit, one healing count) and F7 (in-town futility latch) as open (not done): "more than a few lines with a test". docs/research/primitives-open-20261104.md rank 7 (P5) notes beltfix5 added the stash-gold withdraw chore and counts stash gold in Input.Gold, never drilled live.

## What Changes

Fix F6 and F7 in the town chores with fake-based tests across two builds; no threshold change.

## Impact

Offline only, not verified live.
