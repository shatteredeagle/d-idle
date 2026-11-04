## Why

docs/research/codereview3.md found two medium and two low defects in gate1fix, recoverykit and xpfallback: a target skip bans every route through it (C1), the per-target fail count never decays (C2), a failed ToTown of a recovery kit trip is not counted (C3), and the plan flag copy check always skips (C4).

## What Changes

- C1: routeBan and routeAllowed ignore a nav-fault skip (unreachable, nav-stuck) of a hop that is not the target.
- C2: targetFails remembers the last failure time and decays; it clears when the character stands in the target.
- C3: a failed ToTown of a RecoveryKitReason trip counts in recKitTrips.
- C4: docs/plan/data/progression-normal.json gets the three xp_fallback_full_kit flags; TestMatchesDocs reads builds-endgame/.

## Impact

internal/bot (goal, geargate, areaskip, task, resource), internal/plandata test, one docs data file. Offline only, not verified live.
