## Why

SKMELEEBOT added the plain Attack's blow model to internal/resource (`Input.Blow`, `Input.Foe`), but nothing fills it from the live snapshot, so production decisions are unchanged. This card wires it: the data the model needs, the pure builders, and a fed-versus-blind comparison.

## What Changes

- `internal/data`: `ItemBase.StrBonus` / `DexBonus` from weapons.txt (the class `ToHitFactor`, `Monster.AC` / `Level` / undead flags and the ItemTypes chain were already loaded); `schemaVersion` 3 so an older cache is re-extracted.
- `internal/combat/melee.go`: `meleeBlow` (player stats, wielded weapon, class table) and `meleeFoe` (pack inside the fight radius), nil when an input is missing; `combat.resInput` fills `Input.Blow` / `Input.Foe` only as a pair.
- `internal/resource/sim.go`: `Blind` and `Weak` scenario flags so a replay can hide the model from the policy while the world still rolls it; `wire_test.go` compares both.
- `docs/skmeleewire.md`: gates, before and after numbers, what stayed approximate.

## Impact

Offline Go and docs. Behaviour changes only through the new inputs and only when the item tables, the stat vector, the weapon and the pack are known. Awaits a live early-game soak (drill R1 of docs/research/sk-melee-trace.md decides whether stat 21 / 22 read as assumed). No spec, ADR or glossary change.

## Overlap (declared)

`internal/resource/sim.go` was owned by skmeleebot, which is merged and archived (this branch is based on its merge), so the file is in this card's `owns:` and no open card overlaps. The edit is two scenario flags, `Blind` and `Weak`, and the input assembly in `world.input`.
