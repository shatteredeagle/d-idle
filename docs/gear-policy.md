# Gear policy: the player decides equipment (TASK gearplayer)

User decision 2026-11-10 ("Fully player-only"): the bot never makes equipment decisions. Equipping, swapping and buying equipment belong to the player. Gear Sets (bot-managed equipping) may come later as an unlockable feature; the upgrade advisor (TASK-213, suggestions only) is the supported direction. Tags: V verified by the offline test named, U unverified (needs a live run), S secondary.

## The knob
`resource.gearDecisions` (docs/knobs.md): `0` = player (default), `1` = bot (legacy). The brief called it `gear.decisions`; the knob groups are a closed set, so it lives in `resource`. Go: `env.ResourceTuning.BotDecidesGear()`.
- `cmd/d2bot/gearpolicy.go` (`withGearPolicy`) lays `1` under the config for a measured or drill run (`D2BOT_RUN` set) or `D2BOT_GEAR=bot`; an explicit `resource.gearDecisions` in the config wins. The Gear Sets unlock will set the same knob (hook: the policy sheet row, not yet written).
- Library and test default is player. Tests that exercise the old paths set `GearDecisions = 1` (helper in each package's tests).

## What the bot does and does not do
| path | class | with player |
|---|---|---|
| `bot.gearStep` (`leveler.Equip`, `leveler.Shop`: better item from the pack, vendor gear) | a | does not run; only `adviseGear` (a read) |
| `town.Rebuy` (poverty rebuy: weapon, armor, helm, boots, gloves, belt, shield) | a | gear skipped; the potion part still runs |
| `town.EquipFound` and the `equip` chore, `needWeaponFound` trip reason | a | off |
| `town.buyBelt` (belt purchase and wear) | a | off |
| Amazon bow path: buy a bow or quiver, `wearRanged`, `wantsBow` trip | a | off; ammo for the worn launcher or thrown stack is still restocked (consumable) |
| `resource.GearGate` rebuy (`CanRebuy`) | a | never; a missing weapon or body armor adds no points and never gates, it raises risk to medium with the reason "no weapon equipped (yours to equip)" (amber on the readiness line) |
| potions, scrolls, tomes, keys, ammo, antidotes | b | unchanged |
| repair | b | unchanged, manual (docs/repair.md) |
| sell and stash by the player's marks, pickup (`loot`) | c, d | unchanged (the pickit still keeps a weapon or armor for an empty slot; picking up is not equipping) |
| the player's own equip requests and pins (docs/inventory-ui.md) | player | unchanged: `leveler.PlayerEquip` runs at the gear step |
| mercenary gear (`internal/merc`) | n/a | not the hero's equipment; unchanged (U: the user may want the same rule, say so) |
| `hand.EnsureMainSet` (press `w` back to the main weapon set) | n/a | unchanged: it undoes a game-side set swap, it chooses nothing |

Tests: `internal/resource` (TestPlayerGate), `internal/town` (TestPlayerPolicy*), `internal/bot` (TestGearStepPlayer*), run for Paladin and Sorceress.

## Advice and the onboarding prompt
`bot.Runner.adviseGear` (gear step, in town) reads `leveler.SwapsFor` (pure) into a cache; `invdesk.Config.Advice` returns the text for a pack item ("Hand Axe: better than your Short Sword in the weapon slot") and `invdesk.Config.Prompt` fills `View.Prompt`. The cache exists so the desk never calls the leveler while it holds its own lock (the Desk.View deadlock, see below). The log line `gear advice` appears when the text changes.

Prompt text (S: a new hero starts with weapon and shield already worn, V from charstats per the SME: Amazon Javelin and Buckler, Barbarian Hand Axe and Buckler, Necromancer Wand, Paladin Short Sword and Buckler, Sorceress Short Staff, Assassin Katar and Buckler, Druid Club and Buckler; plus 4 minor healing potions, 1 town portal scroll, 1 identify scroll; no armor, helm or belt):
- Normal: "Your hero wears its starting gear. The bot never changes equipment: equip upgrades yourself on the Inventory page, it only suggests."
- A class that needs a weapon with none worn (the player took it off): "No weapon equipped. Equip one on the Inventory page; the bot will not do it for you."

So a fresh hero is not unarmed: it fights with the starter weapon, wears no armor until the player equips some, and the page tells the player what to do. U live: how the watch page renders `View.Prompt` (the field is in the API; the page text is a follow-up for the invui owner).

## Unarmed and lightly equipped heroes
Fights use whatever is in the right hand; with nothing the hero punches (`combat/melee.go` reads the equipped weapon and tolerates nil, U live). The gear gate cannot close the field over a missing slot (V `TestPlayerGate`), the town does not trip for it, and a build rule that needs a shield (a Smite-style rule) only reads what is worn; it never asks the bot to fix it. U live: a 5 minute unarmed Paladin run is not done (no game).

## Desk.View lock
`Desk.View` holds `d.mu` and calls `leveler.CheckPlayerEquip`, which reaches `Desk.Pinned` (the same lock). The fix is agent/standstill (card for that deadlock); this branch does not duplicate it. The advice added here reads a cache and cannot re-enter the desk.

## Unlock hook for Gear Sets
Set `resource.gearDecisions` to `1` (a policy-sheet row bound to the unlock). Nothing else needs to change: every path above reads `BotDecidesGear()`.
