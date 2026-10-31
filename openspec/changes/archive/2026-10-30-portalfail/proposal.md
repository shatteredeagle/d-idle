## Why

Fresha i5 soak (2026-10-30 10:52-11:44, clvl 4-6) logged 159 `town portal attempt failed` and 53 `safety action failed` lines. Cause: the character carried an EMPTY Tome of Town Portal (tbk, no stat list), the fight code counted the tome as an available portal, and every low-life tick re-ran three useless clicks, about every 7 s for 12 minutes. Separately the shop topped scrolls up to 5 with the gold needed for potions.

## What Changes

- combat treats an empty tome as no portal (Safety, openPortal), and backs off after a failed cast
- one `portal: attempt N failed because <reason> scrolls=a->b` line per failed attempt
- shop caps spare scrolls at 2 until gold is twice the trip floor
- docs note in docs/earlygame.md
