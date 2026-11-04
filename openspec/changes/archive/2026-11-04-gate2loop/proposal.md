## Why

Gate 2 seed 13 (run gate2-s13-20261103-2211-g2v) stalled from about 01:20 EST on 2026-11-04: no kill and no XP for thousands of virtual minutes, 800+ bot restarts. The bot log shows a loop: field area with an empty tome -> "no town portal: walking back to town" -> "Entered Rogue Encampment" -> "save point: saving and re-entering the game" (level 16) -> Save and Exit -> d2d restart -> back in the field -> repeat, never shopping.

## What Changes

Offline root-cause analysis from the log, a fake-based regression test reproducing the loop, a minimal fix that lets the town trip finish (shop, refill tome) before the next save point, and a doc.

## Impact

Offline only, not verified live; the fix needs a Gate 2 rerun.
