## Why

User decision 2026-10-30: "raise the limit to as many as the worker host can handle before you hit performance issues." Agents starved on the 4-slot d2d pool (one waited 100+ min).

## What Changes

- measure the scaling knee of concurrent hidden d2d + d2bot loops on the worker host
- tools/d2dslot.sh: pool size from D2D_SLOTS (ceiling, max 12), admission control for slots >= 4, port block 9100+n (bot) and 9200+n (view)
- internal/d2dreplay reads pool size and port map from ~/d2d-lab/slots/pool.json or D2D_SLOTS
- docs/process/worker-compute.md: pool is a ceiling, admission control protects the host; the floating soft cap stays the user's decision
