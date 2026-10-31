## Why

Gate v2 (openspec/changes/gatedelta) needs two life proxies from the bot's soak minute line.

## What Changes

- minute line gains `dmgTaken` (running counter since bot start, raw HP lost, sum of negative life deltas) and `hpMinPct` (lowest life percent with a hostile in fight range during the minute, 100 when no fight)
