# Digest: combat

> status: current (TASK-310 digest; the area doc is the authority).

## Purpose
Fighting, buffing, survival (potions, town portal, chicken) and the single death transition. The fighter walks through a consumer-side `Mover` and never owns pathing.

## Key files
- `internal/combat/` (fighter, class profiles in `classdata`, cast loop, reach)
- `internal/d2dcombat/` (d2d backend input and meter)
- `docs/combat.md` (API, VERIFIED LIVE sections, dead ends, crowd rule, lockstep spin guard)
- `docs/contracts/combat-loop.md`, `docs/contracts/death-rule.md`
- `docs/d2d-combat.md`, `docs/d2d-casts.md`, `docs/combatspin.md`

## Invariants
- A failed walk costs at least one poll tick; ten failed walks may never spin without the clock moving (docs/contracts/combat-loop.md).
- No retreat while pinned (docs/contracts/combat-loop.md).
- The fight loop has a no-progress watchdog that logs exactly one `combat no progress, holding` line (docs/contracts/combat-loop.md).
- A champion group at low level is marked Avoid (docs/contracts/combat-loop.md).
- `Fighter.Died` is the one death transition; the bot, not combat, emits the Death goal event (docs/combat.md, API summary).
- Class and build specifics are data in the combat profile; write the test for a new rule against two builds (AGENTS.md, Build flexibility).
- Knob values are cited from docs/knobs.md, not copied.

## Gotchas
- The skill-tree (T) route for hotkey binding does not bind anything; bind through the popup and verify behaviourally (docs/combat.md, Dead ends).
- Esc with nothing open opens the game menu; close the popup by clicking the chosen icon (docs/combat.md, Dead ends).
- Kiting every target separately burned the budget: the budget is per fight burst, and a cast-gap sleep starved survival checks (docs/combat.md, Dead ends).
- Under lockstep a tight failing loop can spin; see the lockstep spin guard (docs/combat.md, last section; docs/combatspin.md).
- The "G1/G2 check" obligation for shared code is retired; verify on d2d through the gate ladder or say U (docs/combat.md status banner, docs/decisions.md).

## Read next
- [../combat.md](../combat.md)
- [../contracts/combat-loop.md](../contracts/combat-loop.md)
