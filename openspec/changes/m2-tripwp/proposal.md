## Why

xpgap measured (24 virtual hours, 85 town trips): a trip costs about 4 minutes of which 2.4 are the walk back to the field, 14 percent of all time is `Returning to the field` at 11 kills/h, and chicken leave-game restarts are 21 percent of the long kill gaps at 129 s each (docs/d2d-xp-gap.md). The bot walks out of the Rogue Encampment even when the target area's waypoint is active and leaves the game on a chicken when a portal would do. Both are nav and trip policy, not economy.

## What Changes

- internal/town/trip.go, internal/bot/waypoints.go: leave town by the waypoint when the target area's waypoint (or the nearest activated one on the way) is active; walk only when none is; return through the open portal when one exists.
- Chicken: portal to town when a scroll or tome charge is held, leave-game only without one (the restart path stays for the no-portal case).
- A paired before/after per docs/ab-design.md (8 seeds per arm, 2 virtual hours): `Returning to the field` share and restarts per hour.
- Delta: early-game-policy ADDED `Leave town by waypoint`.

## Dependencies

After basicsfix is archived (internal/town) and m2-abdesign is archived (the protocol), and after soak-slot:3 is free.
