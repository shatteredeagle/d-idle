## Why

Fresha first contact with Akara and Charsi failed three hover clicks while the greeting speech played, skipping the shop chore on two trips and burning 12 to 34 seconds per visit.

## What Changes

Re-read the NPC position before every click, keep waiting for the menu when a hover misses during the greeting, shorten the greeting slice to 4 seconds and cap an interact at 3 seconds. Root cause: hover is dead while the speech plays, not a stale position (from the log).
