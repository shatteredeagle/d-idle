## Why

Follow-up of card mercpolicy: the field rules (wait for the Rogue after a portal or waypoint, leave when the crowd rule fires with her down, a healing potion for her below 50 percent) and the persistence of the Lut Gholein first-visit flag were written and unit-tested but not connected to the running bot.

## What Changes

- Wait: `bot.MercWait` hook (internal/bot/mercwait.go), called after `BackToField` of a town trip and after a waypoint hop in `travel`; `mercWire.WaitForMerc` holds still up to 6 s until a living merc is within 12 subtiles (`merc.ShouldWaitForMerc`), logs `merc: waiting for the merc why=<portal|waypoint>` and `merc: wait over waited_ms=<n> near=<bool>`.
- Leave: `Fighter.mercDownLeave` (internal/combat/mercfield.go) inside `Fighter.Safety`, after the ordinary rules say nothing: crowd (combat.crowdN) with the merc owned and down (`merc.LeaveWithMercDown`) gives a town portal when the cast can succeed (fewer than crowdAdjacentMax adjacent), else a retreat step, but NOT a retreat while a runner is close (`runnerGuard` knob, same test as the kite). Log `merc: down in a crowd, leaving action=<a> crowd=<n>` at most once a minute.
- Potion: `Fighter.mercPotion` in the fight loop after the survival step: `merc.ShouldGiveHealing` (below `merc.potionLifePct`, non-trivial fight = 2 or more hostiles in the fight radius, a healing potion in the belt), gap 4 s, through the semantic input `Belt(col, true)` = d2d `cmd belt N merc` (the verb exists in internal/d2dctl; see docs/merc-wire.md). An input without the verb logs `merc: potion wanted (no verb)` at most once a minute.
- The Fighter seam is wrapped by `installMercSeam`: with `merc.enabled` 0 nothing is installed (`Fighter.MercState` nil, `bot.MercWait` nil), with 1 the Fighter reads the running wiring.
- First-visit flag: `<instance state>/merc.json` (`{"lutDone":{"<character>":[difficulties]}}`, same one-small-JSON-per-instance pattern as the counters file), read once per character, written atomically at each final decision; a decline for gold is not written.

## Overlap (declared)

Cards mercpolicy (cmd/d2bot/mercwire.go, mercwire_test.go, docs/merc-wire.md, docs/botlogs.md) and its archive: this card continues them and is based on their branch. internal/combat/fighter.go and fight.go (a field and a call each), internal/bot/goal.go (two calls) and cmd/d2bot/d2dbackend.go (one line setting the state path) are touched by other cards for unrelated reasons; the edits are additive.
