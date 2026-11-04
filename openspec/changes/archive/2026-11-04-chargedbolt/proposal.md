## Why

The 2026-11-03 Charged Bolt run on d2d could only read the sum of the bolts that hit per cast (`monsters` life in whole points), so no verdict against the per-bolt table was possible.

## What Changes

- A per-bolt method with existing verbs only: step one frame at a time after `debug cast 38 unit <id>`, read `monsters` and `debug missiles` each frame, attribute the life drop to the bolts that left the list in that frame.
- Evidence docs/probes/sksmoke/2026-11-04-SK-cb.txt, .json, -raw.txt, scripts, and docs/chargedbolt.md. Verdict PASS at levels 1, 4, 8, 13, 20 (compared to table on d2d, not game.exe).

## Impact

Docs and probe scripts only; no Go change, no fork edit. Used slot 3 of the worker host, nothing else.
