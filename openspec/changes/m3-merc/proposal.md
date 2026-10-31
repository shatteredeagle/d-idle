## Why

The free Rogue is coded (merc-policy spec, docs/merc-wire.md) behind `merc.enabled` default 0 and was never run for hours on d2d; the route guard needs a living merc to enter the Monastery road below its level floor, so without a merc that works for hours the Act 1 clear waits for clvl 18 or more (d2-sme 2026-10-31). This card makes the claim, the keep-alive and the revive work on d2d for hours and turns the knob on for d2d.

## What Changes

- internal/merc: revive per the revive rule (merc-policy), stop rule, potions to the merc below the knob; the claim once after Blood Raven; log lines with price and gold delta.
- internal/bot/resource.go: `merc.enabled` default 1 on the d2d backend (Game.exe default unchanged, frozen).
- docs/merc-d2d.md: 2 seeds x 4 virtual hours from clvl 9 (cheat-levelled, ledgered) after Blood Raven: uptime, revives, gold, what killed her.
- Delta: merc-policy ADDED `Merc uptime in Act 1`.

## Dependencies

After grill 0003 (docs/roadmap.md decision 4: revive budget and stop rule) and after soak-slot:3 is free.
