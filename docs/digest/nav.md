# Digest: navigation

> status: current (TASK-310 digest; the area doc is the authority).

## Purpose
Walking, exits, waypoints, exploration and roaming on the current level. `nav.Navigator` implements the `Mover` contract (GoTo, GoToUnit, ExitTo, TakeWaypoint) used by combat, town and loot.

## Key files
- `internal/nav/` (A*, exits, explore, covview, mouth gate, per-act hints)
- `internal/d2map/`, `internal/d2dworld/` (map and world sources)
- `docs/nav.md` (API, design, dead ends, NAVFIX/NAVSTALL/NAVSTUCK/NAVMEM sections)
- `docs/roamspacing.md` (roam coverage and detour marking), `docs/d2d-walk.md`
- `docs/fixnav.md`, `docs/navexit21.md`, `docs/navexit22.md`

## Invariants
- One goroutine drives a Navigator; nav is the caller of `Tracker.Update` and calls `World.Update()`; never call it concurrently (docs/nav.md).
- Everything sleeps on `e.Clock`, never wall time (docs/nav.md, API summary).
- Use `GoToUnitOf(ctx, type, id)`; unit ids are only unique per type (docs/nav.md, Notes for other agents).
- `ErrDead` is the hand-over signal to the death logic; nav does no corpse pickup (docs/nav.md).
- A failed walk must not return instantly without costing a tick (docs/contracts/combat-loop.md).

## Gotchas
- Walking to the nearest reachable point when the goal is unknown traps in local minima; plan optimistically through unknown cells (docs/nav.md, Dead ends).
- Detect new terrain with `KnownCount`, not the tracker grid `Version` (it changes every tick) (docs/nav.md).
- Skipping waypoints by distance alone cuts corners into walls; line of sight is required (docs/nav.md).
- Stalls leaving areas and an unbounded memory growth were fixed offline; read the NAVSTALL, NAVSTUCK and NAVMEM sections before touching the stuck ladder or the map cache (docs/nav.md).
- Exit-tile avoidance and pack stalls (PORTALNAV) and island hopping while teleporting have their own sections in docs/nav.md.
- Area names from memory are verified only for early areas; tag anything else U (docs/watch.md, Gotchas).

## Read next
- [../nav.md](../nav.md)
- [../roamspacing.md](../roamspacing.md)
