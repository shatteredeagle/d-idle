## Why

Fresha (i5, Game.exe backend) died twice to Carvers in Dark Wood on 2026-10-30 and the second autopsy carried the death-rule tag "BUG: 2 healing potions in the belt and none drunk". Find the cause and fix what is shared logic. Evidence and verdict: docs/potion-bug.md.

## What Changes

- drink between town portal tries; log every skipped drink with its reason
- no gold sweep into a pack that was at the death spot
- autopsy shows field seconds and seconds under the drink line next to the BUG tag
