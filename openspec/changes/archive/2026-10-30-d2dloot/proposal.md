## Why

The d2d wired bot never picked anything up (autoloot is off, Loot seam nil): 0 gold, 0 potions bought, clvl 1 forever (docs/d2d-wire.md). The real looter must run on d2d.

## What Changes

- internal/d2dloot: ground reads adapted for internal/loot (gold amount, quality guess from the label, position fallback)
- cmd/d2bot/d2dloot.go: the Loot seam (one line in d2dwire.go)
- docs/d2d-loot.md: what is verified, live numbers
