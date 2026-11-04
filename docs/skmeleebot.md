# SKMELEEBOT: the plain Attack in the resource policy

> status: current (2026-11-01). Offline: no game, no d2d slot. Source: docs/research/sk-melee-trace.md (B1 to B4), oracle tools/emu/sk_melee.py. Tags: V = verified (run or read, as the trace says), I = inference, U = unverified.

## What changed and why

| Finding | Before | After | Where |
|---|---|---|---|
| B1 simulator melee | every swing lands for a flat 3 | a swing rolls to-hit (`HitChance`, FUN_0057d9b0, V run) and a damage range in 256ths (FUN_0057b420 reading, V run); stock scenario: 95 % of a mean 3.297 = 3.13 per swing; replay of the stock scenario: 23 kills / 37 swings (flat) vs 22 kills / 28 swings | `melee.go`, `sim.go` |
| B2 `WeakPack` | level only | with `Input.Blow` and `Input.Foe` known the pack must also be hit at least `WeakMinHitPct` (50, U policy choice): AR 95 vs defense 15, same level: 86 % stays weak; vs defense 100: 48 % is not weak (table test); no model: unchanged | `types.go` `weak()`, `attack.go` |
| B3 undead rung | no undead term | a Blunt weapon vs an all-undead pack raises the melee rung's pack limit by the damage gain: 1-5 staff, p 10 to p 60: 3.297 to 4.797 per landed blow (x1.455), so the default limit 3 becomes 4; living foes and edged weapons keep 3 | `types.go` `meleePack`, `attack.go` line for the melee rung only |
| B4 | | early d2d melee soaks carry D1 (the physical floored to whole points, about 0.5 less per landed hit) and D2 (no Blunt +50 % vs undead): d2d kills slower than game.exe at clvl 1 to 11 with staff, mace or scepter. Docs only; no d2d change here | this note |

## Interface

- `Blow` (attacker: AR, AR percent, level, stats 21/22, stat 111, Str, damage percent, stats 17/18, `Blunt`) and `Foe` (defense, level, undead) are filled by the caller from the character's stats and the data tables: `AttackRating(stat19, dex, toHitFactor)`, `StatDmgPct(gear, str, dex, strBonus, dexBonus, mastery)`, `IsBlunt(isA, type)` with `data.DB.IsA` as the chain walk (ItemTypes `blun`; staff, scepter and wand reach it, trace D). No class, skill or weapon name is in the code.
- `Input.Blow` or `Input.Foe` nil means unknown: behaviour is exactly as before. Cross-build check: a test uses a staff caster and a sword fighter with different tables.

## Tests and oracle

Expected values in `internal/resource/melee_test.go` come from the oracle's own reading functions (`hit_chance`, `dmg_reading`, `muldiv`), executed offline without the emulator on the same inputs (the file imports `emu` only for the game.exe part; the functions were exec'd with a stub). The game.exe differential itself is the trace's V run; it was not re-run here (needs the worker host and the game binary).

## Stayed approximate

- Crit, deadly strike, crushing blow, open wounds, leech, IAS and swing speed are not modelled (the simulator keeps its 1.0 s swing).
- Stats 17/18 default to 0 (reading A, trace Q3, U until drill R1); the model has the fields.
- Stat 120/121/122/180 (undead/demon/monster-type gear terms) and AR terms 115/116/123/124/179 (D3) are not modelled.
- Stock scenario inputs (to-hit factor 5, Blood Moor foe defense 15 and level 3, staff 1-5) are U nearest defaults; the 50 % `WeakMinHitPct` and the "limit scales with the gain" rule are policy choices, not game rules.
- Production wiring: `combat.resInput` does not yet fill `Input.Blow` / `Input.Foe` (needs weapon StrBonus / DexBonus, which internal/data does not load, and the class ToHitFactor lookup). Until then B2 and B3 are active in the simulator and unit tests only.

## Shared-file lines for the merger

- docs/BACKLOG.md: `- skmeleebot (card [skmeleebot](archive/openspec-index.md#skmeleebot), offline): SKMELEE B1 to B4 done: the simulator's melee rolls to-hit and damage (internal/resource/melee.go), WeakPack has the to-hit term, the melee rung scales its pack limit with the Blunt-vs-undead gain, B4 noted in docs/skmeleebot.md; open: fill Input.Blow / Input.Foe from the live snapshot (weapon StrBonus / DexBonus not loaded).`
