## Why

The free Rogue is coded (merc-policy spec, docs/merc-wire.md) behind `merc.enabled` default 0 and was never run for hours on d2d; the route guard needs a living merc to enter the Monastery road below its level floor, so without a merc that works for hours the Act 1 clear waits for clvl 18 or more (d2-sme 2026-10-31). This card makes the claim, the keep-alive and the revive work on d2d for hours and turns the knob on for d2d.

## What Changes

- internal/merc: revive per the revive rule (merc-policy), stop rule, potions to the merc below the knob; the claim once after Blood Raven; log lines with price and gold delta.
- internal/bot/resource.go: `merc.enabled` default 1 on the d2d backend (Game.exe default unchanged, frozen).
- docs/merc-d2d.md: 2 seeds x 4 virtual hours from clvl 9 (cheat-levelled, ledgered) after Blood Raven: uptime, revives, gold, what killed her.
- Blood Raven bailout: after 3 failed attempts or at clvl 12 without the kill she counts as skipped for good (`BloodRavenSkipped`) and the paid Rogue fallback applies.
- Revive only when gold minus the kit reserve >= `merc.reviveGoldMult` (2) x price; `merc.reviveMaxPerHour` (2), `merc.stopGoldMult` (5), `merc.stopDeathsPerHour` (3) unchanged.
- Keeper versus throwaway: a build's target merc or a merc wearing target gear is a keeper (always revived); a throwaway is re-hired when the hire price is below the revive price. Until builds define a target merc the bot always revives.
- Delta: merc-policy ADDED `Merc uptime in Act 1`, MODIFIED `Revive and stop rule`, ADDED `Blood Raven bailout` and `Keeper and throwaway mercs`.

## Open questions (parked by grill 0003)

- Build merc targets (target merc and gear per build): needed before the keeper rule has any keepers.
- Live probes on d2d or the original: dead-merc forfeit of the free Rogue, one revive quote, 2 to 3 hire windows, merc Str and Dex own-stats rule, merc state after quit and reload. Revive and hire prices are DERIVED, never seen live.

Grill 0003 ([ADR 0011](../../../docs/adr/0011-readiness-power-vs-danger.md), accepted 2026-10-31) sets the rules this card implements: revive only from gold above the kit reserve, a Blood Raven bailout, and keeper versus throwaway mercs.

## Dependencies

After soak-slot:3 is free (grill 0003 is done: ADR 0011).

