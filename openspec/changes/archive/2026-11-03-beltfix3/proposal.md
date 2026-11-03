## Why

Gate 1 run gate1-20261102-2210 still failed BELT (37 virtual minutes) and NOKILL (11): the character visited town six times with no healing potion and 300+ gold and left without shopping, and from Dark Wood, Stony Field and the caves there was no way home (docs/research/beltfix2-analysis.md).

## What Changes

- Shop on every town arrival with no healing potion (short 5 s pause instead of the resupply cooldown).
- Walk the level chain home when there is no portal source and no active waypoint; the resource trigger and kitTrip agree through a wider walkableHome.
- A cast cut after the click is reported as "source maybe spent", and the next portal call reuses the cut cast's portal.
- BackToField retries the town portal once before the waypoint and the walk.

## Impact

internal/bot/goal.go, internal/town/trip.go, internal/resource/town.go, cmd/d2bot/d2dwire.go and their tests. No spec delta.

Overlap (declared): cmd/d2drill/portal_copy.go (owned by m1-liveproof, glob cmd/d2drill/**) is a verbatim copy of the portal verb in cmd/d2bot/d2dwire.go (TestPortalCopyInStep); this card refreshes only that copy. The drills card owns the rest of cmd/d2drill.
