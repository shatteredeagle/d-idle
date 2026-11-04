## Why

docs/research/primitives-open-20261104.md ranks the tome stat-70 dropout first (P10/P12, S/A): in the field the tome's stat 70 reads 0 (`inv-tp-dump counted=0 raw70=no70`) while the town reads 13 to 19, so a false "portal empty" starts a trip loop. docs/tomestat.md fixed one cause (unit-id collision in d2dsrc extras), docs/townshop.md fixed the cost; the cause is still U and nothing was run live.

## What Changes

- Offline audit of every path that can produce a false-empty portal verdict (town.qty, combat.PortalQty, d2denv reader snapshot staleness, tomestat fix coverage), a regression test per found gap with fakes, a fix only for a gap that the test proves, and a live-check recipe (docs/tomefix.md) for the drill.

## Impact

Offline only, not verified live. Policy thresholds unchanged.
