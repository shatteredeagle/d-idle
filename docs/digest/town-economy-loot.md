# Digest: town, economy and loot

> status: current (TASK-310 digest; the area doc is the authority).

## Purpose
Town chores (heal, identify, sell, shop, repair, stash, bank, merc), the trip decision, the pickit and pickup, and the currency registry and ledger.

## Key files
- `internal/town/` (chores, belt plan, hand, panel, sim, merc), `internal/loot/` (pickit, pickup, potion floor)
- `internal/d2dtown/`, `internal/d2dloot/` (d2d adapters), `internal/currency/` (registry, ledger, bank)
- `docs/town.md`, `docs/townshop.md`, `docs/townwedge.md`, `docs/currency.md`, `docs/d2d-town.md`, `docs/d2d-loot.md`, `docs/d2d-economy.md`
- `docs/contracts/town-basics.md`, `docs/contracts/early-game-policy.md`, `docs/contracts/merc-policy.md`

## Invariants
- Identify scrolls only on demand; potions before the portal scroll when poor; bank before the shop (docs/contracts/town-basics.md).
- A repair failure latches: the second visit with unchanged gold and durability skips the step (docs/contracts/town-basics.md).
- No vendor gear that needs more Strength than the character has; rejuvenation potions are capped (docs/contracts/early-game-policy.md).
- Take the free Rogue mercenary and never pay Kashya before the quest (docs/contracts/merc-policy.md).
- Each chore step has a need check; a failing step does not stop later steps; panels are closed at the end (docs/town.md).
- Currencies are data (`internal/currency/registry.json`); vanilla mode creates no ledger file (docs/currency.md).
- Do not patch the d2d fork from the town side; file a fork verb request instead (docs/d2d-town.md, Fork verb requests).

## Gotchas
- Fixed dialog pixel rows do not work; the box moves with the NPC (docs/town.md, Dead ends).
- Vendor buy price is not the base cost column; confirm a purchase by the gold change (docs/town.md, Dead ends).
- Unit ids are unique per type only; the d2input helper of an instance can die, restart it with `tools/game.sh helper iN` (docs/town.md).
- Currency work is offline only, not verified live (docs/currency.md status banner).
- Item class index is the position in the data tables, not a game code (docs/town.md).

## Read next
- [../town.md](../town.md)
- [../currency.md](../currency.md)
