## Why

docs/basics-audit.md E4 is G: the stash gold cap is not handled anywhere (stash.go depositGold has no cap logic), so once the stash is full every deposit comes back `not confirmed` and the stash step fails on each visit for the rest of the character's life. The cap value is unknown (U) and is measured by m1-liveproof; the code change is small and separate from the shop and chores files.

## What Changes

- internal/town/stash.go: when the deposit is refused for a full stash, deposit the difference to the cap and log once; the cap is a constant with its source (m1-liveproof unknown 3, U until measured).
- internal/town/stashcap_test.go: TestStashGoldCapDeposit (partial deposit, one log line, no retry while full).
- Delta: town-basics ADDED `Stash gold cap`.

## Dependencies

After basicsfix is archived (it may touch internal/town/stash.go for bank-before-shop).
