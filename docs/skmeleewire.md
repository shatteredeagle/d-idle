# SKMELEEWIRE: the melee model fed from the live snapshot

> status: current (2026-11-01). Offline: no game, no d2d slot. Follows docs/skmeleebot.md (the model) and docs/research/sk-melee-trace.md. Tags: V verified (run or read), S secondary, U unverified.

## Data

| Input | Source | Status |
|---|---|---|
| class to-hit factor | `data.CharClass.ToHitFactor` (charstats), already loaded | V (loaded and read by `meleeBlow`) |
| weapon StrBonus / DexBonus | weapons.txt `StrBonus`, `DexBonus`: NEW in `data.ItemBase`, `schemaVersion` 3 (an older cache is re-extracted by the stamp) | S (column names from the 1.14 weapons.txt; not read from a real cache here, only the synthetic fixture) |
| weapon type and Blunt chain | `ItemBase.Type`, `DB.IsA` (ItemTypes Equiv1 / Equiv2), already loaded | V (code); the staff -> rod -> blunt chain is I (trace D) |
| foe defense, level, undead | `Monster.AC[diff]`, `Level[diff]`, `LUndead` / `HUndead` (monstats), already loaded | V (code); area-level overrides and bosses not read (U) |
| attack rating, damage stats | player's merged stat vector: 19, 21, 22, 25, 111, 119, 325; Strength, Dexterity, level from `Player` | ids V read in game.exe (trace); that the player's list carries 21 / 22 as the wielded weapon's range is U (drill R1) |

Nothing was unavailable; no number is invented. Not modelled (0): weapon-mastery to-hit and damage (FUN_00645830; skill levels are in the snapshot but the mastery tables are not loaded), stats 17 / 18 (reading A), crit and speed.

## What `combat.resInput` fills, and what gates it

`Input.Blow` and `Input.Foe` are set together, from `meleeBlow` and `meleeFoe` (internal/combat/melee.go). Either is nil, and then so is the other, when:

- the game tables are not loaded (`Env.Data` nil), the player or its stat vector is missing, or the inventory is missing (the wielded item is unknown);
- the class has no charstats row;
- the right hand holds a row outside the item table, an armor or misc row, a ranged weapon (type reaches `miss`, S) or a throwing weapon, or a weapon while stats 21 / 22 are absent;
- no hostile is inside the fight radius, or one of them has no monstats row.

Barehanded with a known inventory is a model (trace D4), not an unknown. The foe is the in-radius hostile the blow is least likely to hit; `Undead` is true only when all of them are undead.

## Decision changes on the stock scenarios

`go test -v -run Wire ./internal/resource`. Replay: 5 minutes, clvl 8 Sorceress, staff 1-5, seed 7, resource policy; "blind" hides Blow / Foe from the policy, the world rolls the same blows (`Scenario.Blind`).

| Scenario | blind: kills / melee swings / casts / retreats | fed: kills / melee / casts / retreats | deaths |
|---|---|---|---|
| stock living (defense 15, level 3) | 22 / 28 / 66 / 2 | 22 / 28 / 66 / 2 | 0 / 0 |
| undead pack | 23 / 37 / 60 / 1 | 21 / 27 / 58 / 0 | 0 / 0 |
| undead, weak by level | 21 / 27 / 58 / 0 | 21 / 27 / 58 / 0 | 0 / 0 |
| defense 100, level 8, weak by level | 22 / 73 / 64 / 0 | 20 / 61 / 61 / 1 | 0 / 0 |

Melee-rung choice with no mana, by awake hostiles (the rung table test):

| Scenario | 1-3 | 4 | 5-6 |
|---|---|---|---|
| stock living | melee | retreat (same) | retreat (same) |
| undead | melee | melee (was retreat: Blunt gain 1.455 lifts the limit 3 to 4) | retreat (same) |
| undead, weak | melee | melee (same: weak already allowed it) | melee (same) |
| defense 100, weak | melee | retreat (was melee: to-hit 48 percent is under WeakMinHitPct) | retreat (was melee) |

Reading: the model changes decisions in exactly the two directions SKMELEEBOT designed (B3 allows one more hostile against undead for a Blunt weapon, B2 stops calling a pack weak when the swings mostly miss). Per-run kill counts move by a few either way because the changed decisions change the pack sequence: the replay is a regression check (no extra death, deterministic), not a measured gain.

## Open

- Live early-game soak with a staff caster and a sword fighter: check the log of the fed Blow (AR, damage range) against the character panel, and drill R1 (are stats 21 / 22 the weapon's range with ED, are 17 / 18 zero). If 21 / 22 read otherwise the gate `st.Has(21/22)` still passes with wrong numbers: the fix is in `meleeBlow` only.
- Weapon-mastery terms; swap-weapon hand; dual wield (second hand 23 / 24).
- A cache extracted before schema 3 and loaded with `data.Load` directly reads StrBonus / DexBonus as 0; `Open` / `OpenCached` re-extract.

## Shared-file lines for the merger

- docs/BACKLOG.md: `- skmeleewire (card [skmeleewire](archive/openspec-index.md#skmeleewire), offline): Input.Blow / Input.Foe are now filled by combat.resInput from the snapshot and the tables (weapon StrBonus / DexBonus loaded, schema 3); nil on any missing input; fed-vs-blind replay in docs/skmeleewire.md; open: live early-game soak, drill R1.`
