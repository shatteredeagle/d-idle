# PALAURA: how an aura applies in game.exe 1.14d (target filter, area search, enemy hit, Prayer's mana and state 85); game.exe vs d2d vs the bot

> status: card palaura, 2026-11-02, offline (no game, no d2d slot, no Go change). Closes the open parts of `docs/research/pal-skill-trace.md` sections 3 and 10 (its D5 and D6 rows in section 6) with game.exe's own code run under upstream's emulator. Not repeated here: the pulse period (50 frames), the ranges and every calc slot (21,760 cases, 0 differ against d2d: pal-skill-trace.md section 3).

Tags: **V read** disassembled (address given; no code pasted); **V run** game.exe's own functions run natively under the emulator by this card (script and count given; hooks and hand-made pieces named in section 9); **D** from the patched 1.14d tables as game.exe loaded them; **S** upstream source read (d2d main at the time of the card, `components/game/fight.cpp`); **I** inferred; **U** unverified. A V run of the game side alone is not a differential: the d2d column is S unless a count against d2d is given. There is none in this card (section 8).

## Summary

- **Area search (V read + V run).** `FUN_0056b7e0` walks the caster's room and its near rooms, takes every unit whose integer subtile position is within the radius by **`dx^2 + dy^2 <= r^2`** (inclusive, euclidean, subtiles), skips the caster itself, asks the filter `FUN_0056b3e0` with the skill's `aurafilter` and calls the callback for each unit that passes. 2,032 random placements: 2,028 as the formula says, the other 4 are units outside the near-room list (radius 43 only).
- **Filter bits (V read, bit table V run).** 0x12003 (every friendly aura) = players and monsters (0x1, 0x2) that are the caster's own, a pet or merc of the caster, or in the caster's party (0x10000), rooms in town skipped (0x2000). A run on a real Paladin unit, a party ally, a merc, pets, enemies, NPCs and corpses: **only the caster, the party ally and the pets / merc of the caster or of the ally pass**, never an enemy, a foreign player's pet, a corpse, a townsperson or a unit beyond the radius. 0xa783 (Holy Fire, Holy Shock, Hurricane) = every living attackable monster (0x2, 0x80, 0x8000) plus players that are hostile (0x1, 0x8000), not in town, **line of sight** (0x200). Holy Freeze's word is 0x4a283: the same without 0x100 / 0x400 and **with 0x40000: prime evil monsters are skipped**.
- **Enemy hit (V run).** Holy Fire / Shock / Freeze: one roll per pulse, `min + rand(max - min)`, i.e. **min .. max-1**, the same roll for every target, then `roll * (100 - eff) / 100` truncated with `eff = resist - the caster's element pierce` (clamped at -100; a resist of 100 or more stays immune). 6,000 pulses of Holy Fire level 1: rolls 256 .. 767 (512 distinct values, 768 never). Nothing happens in town, to the caster's pet, to an NPC, outside the circle or (Holy Freeze) to Andariel.
- **Prayer (V run).** The heal is a direct add to the unit's life clamped at its maximum, done by the callback for **every unit that passes the filter**; mana is paid, and state 85 (`nomanaregen`) put on the caster, **only on a pulse in which at least one unit's life actually rose**; on a pulse with nobody hurt state 85 is cleared and nothing is paid; with less mana than the cost no heal happens at all. State 85 zeroes the regeneration base and the percent bonus and **keeps the flat bonus** (stat 26).
- **d2d (S).** Friendly auras: the player's own stats and a player-only Prayer heal, no mana, no state 85, no filter. Enemy auras: every living monster within `hypot(...) * 5 <= radius`, the roll of `missile_blow` (SKMISS D1: top included), no filter, no line of sight, no town rule, no prime-evil rule.
- **Deviations D1 to D9** (section 7), **bot findings B1 to B3** (section 8), **no PR candidate** (section 8: nothing is proven by an existing d2d differential).

## 1. The area search (FUN_0056b7e0)

Called by the friendly do 65 (`FUN_005cf010` at 0x5cf224), the enemy do 66 (`FUN_005cf3a0` at 0x5cf64e) and others. Arguments: `ecx` game, `edx` caster; stack: x, y (both 0: the caster's own position, `FUN_0045adf0` / `FUN_0045ae20`), radius, filter word, callback, callback argument, skip flag, then a source-file string and line (assert data, not logic). A filter word 0 means 0x583.

| Step | Rule | Tag |
|---|---|---|
| rooms | the caster's room (`FUN_00620bb0`) and its near list (`FUN_00619790`); each room is kept when its bounding rectangle overlaps the square `(x +- r, y +- r)` (`FUN_0056b770`); with filter bit 0x2000 a room is skipped when `FUN_0061ab00` (room is in a town level) says yes | V read |
| walk | every unit in the kept room's unit list (room +0x74, next link unit +0xe8); the caster (`edx`) itself is skipped | V read |
| position | unit type 2, 4, 5: path +0xc / +0x10; others: the path's integer subtile words (`FUN_00648900` / `FUN_006488c0`); the centre is the caster's integer position too | V read |
| distance | `FUN_006492a0` = `dx*dx + dy*dy`; a unit is dropped when `r*r < d2`: **inclusive**, euclidean, in subtiles (the range calc's unit) | V read, **V run** |
| filter | `FUN_0056b3e0(caster, unit, filter)` must pass (section 2) | V read, V run |
| skip flag | with the flag set (the friendly do sets it, Holy Fire does not) a monster whose MonStats flag dword has bit 27 is dropped: `FUN_00457490(class, 0x1b)`; bit 27 is the column **noAura** (5 rows: classes 340 to 344) | V read + **D/V run** (palaura_flags.py) |
| callback | `callback(ecx = {game, caster, count, argument, r*r}, edx = unit)`; a nonzero answer counts | V read |

**Q1. What is the shape?** A disc of integer subtiles, radius in subtiles, inclusive, limited to the caster's room and its near rooms. **V run** (`palaura_area.py radius 5`): per radius (6, 7, 15, 16, 18, 25, 34, 43) 127 units at random offsets in the square `[-r-3, r+3]^2` plus the seven boundary points (r,0), (-r,0), (0,r), (0,-r), (r+1,0), (0,r+1), (r,1), with the callback hooked to record the unit; once with filter 3 (monsters) and once with 0x12003 for owned pets: **2,032 units, 2,028 agree with `d2 <= r2`, 4 do not**, and all 4 are units whose room is not the caster's room or in its near list (radius 43, west side); the printout says so per unit. Units exactly on the circle (d2 = r2) are found; one subtile outside are not.

**Q2. Who is passed?** Self is not passed to the search (the do applies the caster's own state first, `FUN_0056b740` at 0x5cf143); the others as in section 2. The merc is an ordinary monster whose owner block names the caster; pets likewise; party players are the filter's party test.

## 2. The target filter (FUN_0056b3e0)

Jump table at 0x56b6ac (V run: five entries 0x56b3ff, 0x56b43c, 0x56b4a0, 0x56b4b3, 0x56b4dd for unit types player, monster, object, missile, item). Bits (the word is `Skills.txt aurafilter`, record +0x50):

| Bit | Test | Tag |
|---|---|---|
| 0x1 | unit is a player (mode 0 or 0x11 rejected unless 0x1000) | V read |
| 0x2 | unit is a monster (mode 0 or 12 rejected unless 0x1000) | V read |
| 0x4 | monster must be undead (`FUN_0063e990`) | V read |
| 0x8 / 0x10 / 0x20 | missile (class record flag) / object / item allowed | V read |
| 0x80 | unit flags +0xc4 bit 2 set ("attackable"): townspeople lack it | V read + V run |
| 0x100 | unit's room is not a town (`FUN_0061ab00`) | V read |
| 0x200 | **line of sight**: `FUN_0064e260(room, caster xy, unit xy, 4)` must answer 0. 76 of 1,200 units in range rejected in one wall-rich area; best fit of a straight line crossing a collision cell with bit 0x4: 1,183 of 1,200 | V read + V run; the mask is I |
| 0x400 | unit flags +0xc4 bit 3 set | V read |
| 0x1000 | corpses only (mode 0x11 / 12) | V read + V run |
| 0x2000 | search skips rooms in town (in `FUN_0056b7e0`) | V read + V run |
| 0x4000 | monster must **not** have MonStats flag bit 6 = **boss** | V read, D (palaura_flags.py: bit 6 equals the boss column on all 734 rows) |
| 0x8000 | `FUN_00554200(caster, unit)`: the unit's root owner differs and is not allied to the caster's (see below) | V read + V run |
| 0x10000 | `FUN_00554de0(caster, unit)`: same root owner, or same party (both players, same party id, not 0xffff), or allied; and a monster with the **npc** flag (bit 8) is refused | V read + V run |
| 0x20000 | `FUN_00622c40(caster, unit, 0)` (U: only that players answered 1 and merc / pets 0 in the run; no Paladin aura uses it) | U |
| 0x40000 | monster must **not** have MonStats flag bit 7 = **primeevil** (`FUN_0063edc0`) | V read, D (bit 7 equals the primeevil column, 12 rows) |
| 0x80000 | unit must not have state 0x56 | V read |

Relations. Root owner: a monster's owner (`FUN_0058f0d0`, from the AI struct block game / owner id / owner type at +0x28 / +0x2c / +0x30 via the unit hash `FUN_00552f60`), a missile's owner (`FUN_00552fd0`), else the unit. `FUN_00554de0` is 1 for the same root, for two players in one party (party id from the game's party list, `FUN_00540710`), else `FUN_00650d70` = alignment equal and not 1 (`FUN_006259b0`: stat 0xac of the unit's state-0x69 list; player 2, ordinary monster 0, none 0). `FUN_00554200`: 0 for the same root, two players: `FUN_0055b300(a, b, 8)` (hostility; none in the run), else the opposite of the alignment test.

**Skills.txt words in use (V run, `palaura_filters.py`, D):** 0x12003 for Might, Prayer, Resist Fire (98..100), Thorns, Defiance, Resist Cold, Blessed Aim, Cleansing, Resist Lightning, Concentration, Vigor, Fanaticism, Salvation; 0x12001 Meditation (players only: no merc, no pets); 0xa783 Holy Fire, Holy Shock, Hurricane; **0x4a283 Holy Freeze**; 0xa583 Conviction (no line of sight bit); 0xe786 Sanctuary; 0x1102 Redemption (corpses); 0x10103 the Druid auras; and others (the script prints all 18 words).

**V run (`palaura_area.py filter 5`, a Paladin player, a party ally, a dead party player, an unrelated player, a merc, three pets (own, ally's, foreign), enemies, Andariel, Akara, Warriv, a zombie, all within 6 subtiles, radius 20):**

| Unit | 0x12003 | 0xa783 |
|---|---|---|
| caster | not passed (the search skips it) | not passed |
| party ally player | PASS | - |
| unrelated player | - | - (no hostility record) |
| dead party player | - | - |
| merc (owned monster) | PASS | - |
| caster's pet | PASS | - |
| pet of the party ally | PASS | - |
| pet of an unrelated player | - | - |
| enemy monster, zombie, Andariel | - | PASS |
| dead enemy monster | - | - |
| townspeople (Akara, Warriv) | - | - (no flag 0x4 at +0xc4 bit 2) |

Each single bit on top of 0x3 was also run (printout of the same script): 0x1000 only corpses; 0x4000 drops Andariel; 0x40000 drops Andariel; 0x10000 passes the merc, pet, party pet and party ally only.

## 3. What the callback does (FUN_005cedc0), and the friendly do (FUN_005cf010)

- The do reads the six `aurastat` ids and calcs (as pal-skill-trace.md), applies the caster's own state (`aurastate`, +0x80) with the stats through `FUN_0056b740` (callback on the caster), then runs the area search with the target state (`auratargetstate`, +0x82), the skill's `aurafilter`, radius = `aurarangecalc`.
- The callback puts the state on the unit with a length reaching past the next pulse, then for each stat with a nonzero value: stat 0x6e goes to its own helper `FUN_005cec90` (U, not read), otherwise **a stat whose ItemStatCost flag byte (+4) has bit 4 set is added to the unit's current value clamped at its maximum** (`FUN_00627150` after a read of the old value, maximum row +0x32); any other stat goes onto the state's stat list. The shared counter (`ctx +0x40`) rises only when a stat was applied or **the direct value really changed**.
- **V run (`palaura_apply.py friendly 98 1`, Might level 1, mana 0):** states and stats land on the caster, the party ally, the merc and the pet (damagepercent stat 25 = 40 = the table's 40 + 10 (L-1) at level 1); not on the unrelated player, a pet 40 subtiles away, or the enemy. **Might works with 0 mana** (cost 0: `0 < 0` is false), so SKMANA D2's refusal at 0 mana is a d2d-side floor in the check only.
- **V run in town (`palaura_apply.py town`):** Might reaches the caster only (the ally, merc and pet stay without); Holy Fire level 5 deals 0 to a rat 2 subtiles away. The caster's own state is applied before any town test, so **Prayer still heals the Paladin himself in town**.

## 4. The enemy hit per pulse (FUN_005cf3a0 do 66, FUN_005d0920 do 81)

- Do 66 reads the five passive stats (computed only when mana covers the cost), puts the passive stats on the caster (`FUN_0056b740`), **returns at once when the caster's room is in town** (`FUN_0061ab00`), builds the target state's stats (computed only when mana covers the cost), then calls `FUN_0056e0c0` **once**: it fills a 0x70-byte hit record with the skill's element min / max (`FUN_00644d50` / `00644e40`, mastery flag 1) and the length (`FUN_00644f20`), rolls `min + FUN_0045c3e0(caster seed, max - min)` and ORs the hit flags (0xd, the skill's +0x12e / +0x130 / +0x134). The area search then calls `FUN_005cf2a0` for each enemy that passes: it applies the target state (Holy Freeze: state 44 = the slow and resist stats) and runs the hit (`FUN_0057c6c0`, `FUN_0057cee0`), the same strike chain as every other skill hit (SKMISS).
- **V run (`palaura_apply.py hit <skill> <level> <pulses> [pierce]`, real Paladin unit with a real seed, Quill Rat, a 50 / 75 / 100 % resist class and Andariel in range; the pet, an NPC and a rat outside the circle also present; life in 256ths):**

| Skill, level | Table min .. max (D, FUN_00644d50 / 00644e40) | Pulses | Observed roll (resist 0 rat) | Result |
|---|---|---|---|---|
| Holy Fire 1 | 256 .. 768 | 6,000 | 256 .. **767**, 512 distinct values, mean 515.9 (table mean 512) | all inside; **768 never** |
| Holy Fire 5 | 768 .. 1280 | 100 | 773 .. 1271 | all inside |
| Holy Freeze 5 | 1536 .. 1792 | 100 | 1539 .. 1789 | all inside; state 44 on the rat |
| Holy Shock 5 | 256 .. 8704 | 200 | 278 .. 8672 | all inside |

  Resist: in every pulse of the runs above and of the pierce run (Holy Fire 5, 300 pulses) each target's damage equals `roll * (100 - eff) // 100` for **one** roll shared by all targets (Holy Fire 1: resist 50 and 75: 6,000 of 6,000; resist 100: 0 always; with the caster's fire pierce stat 333 = 40: effective resist = resist - 40, so a resist-0 rat takes **+40 %** and a resist 50 rat 10 % resist: 300 of 300). Rounded instead of truncated explains 3,109 of 6,000: truncation is the rule. Outside the circle, the caster's pet, an NPC: 0 damage in every pulse.
- **Prime evil:** Holy Freeze (0x4a283) did **0 damage and no state** to Andariel in 100 pulses; the same Holy Fire (0xa783) pulses hurt her (her fire resist from MonStats is -50 on the normal table: damage = roll x 1.5; V run, `class 156` line of the printout).
- Which units: monsters that pass 0xa783 (section 2), not in town, with a straight line; **not** the caster's own pets, the merc, NPCs, corpses.

## 5. Prayer: heal, mana and state 85 (V run, V read)

Cost `FUN_00644b10` = `(lvlmana x (lvl - 1) + mana) << manashift`: Prayer level 5 = 448 (256ths) = 1.75 mana (the trace's table). Order in the do: with mana below the cost the stat calcs are skipped (every value 0, so **no heal at all**); otherwise the callback heals every unit that passes 0x12003 (the caster first); then, **if the cost > 0, the caster is a player and the shared counter > 0**: state 85 set, cost paid (`FUN_0056c110`, stat 8); otherwise state 85 cleared.

`palaura_apply.py friendly 99 5 <hurt list> <mana> [state85]` (Prayer level 5: heal per pulse 1536 / 256 = 6 life; every case a fresh game, seed 5):

| Case | Heal | Mana | State 85 afterwards |
|---|---|---|---|
| caster hurt (life 40 of 80), mana 100 | caster +6 | 100 -> 98.25 | on |
| nobody hurt, state 85 on before | none | 100 -> 100 | **cleared** |
| only the party ally and a pet hurt, caster full | ally +6, pet +4 (clamped at its maximum) | 100 -> 98.25 | **on (the caster is not hurt)** |
| caster hurt, mana 1 (< 1.75) | **none** | 1 -> 1 | off |

**Regeneration (V run, `palaura_apply.py regen`: `FUN_005806f0` with EDI = a real Paladin, max mana 100 points, mana 0):** state 85 off: +8 (256ths, the base `max(25600 / 3000, 1)`), with a stat 27 of 100 %: +16; state 85 on: **0** and **0**; with a flat stat 26 of 512: off 520 / 528, on 512 / 512. So the state stops the base and its percent bonus (what Warmth and the character's regen come from) and not the flat bonus.

So: a resting Paladin in Prayer at full life pays nothing and regenerates mana normally; a hurt Paladin (or one with a hurt merc or pet in range) pays 1.0 .. 4.6 mana every 2 s and does not regenerate mana (the flat bonus aside) for that pulse; with less mana than the cost, Prayer neither heals nor blocks the regeneration.

## 6. d2d's version (S, upstream `components/game/fight.cpp`, `Fight::aura_pulse`, `in_aura`, `target_of`)

- Pulse every `max(perdelay, 25) * 40 ms` from the selection (game.exe: `max(perdelay, 5)` frames on the frame grid 1 mod p; equal for the Paladin's 50).
- Friendly do: only aurastat 6 is handled (`min(max life, life + heal)` on the player, always, free); the other stats are the player's own self state (`update_fighters`). No merc, pet or party, no mana, no state 85, no filter.
- Enemy do 66 / 81: `damage = elem_damage(min, max)`, then for every monster with `alive() && in_aura` (`hypot(monster - player) * 5 <= radius`, floats) `land(missile_blow(damage, target, pierce, rng))`. `missile_blow`'s roll is `Rng::range(min, max)` = `min + rand(max - min + 1)` (SKMISS D1), pierce from the player's passive stats, resist as SKMISS. Holy Freeze's stats go through `target_of` / `apply_target_stats` (resist and armor class). No aurafilter anywhere.

## 7. Deviations d2d vs game.exe (game side V run unless noted; d2d side S)

| id | rule | game.exe | d2d | tag | differential |
|---|---|---|---|---|---|
| D1 | who a friendly aura reaches | caster, party allies, the merc and pets of the caster / allies within `16 + 2 (L-1)` subtiles (Might, Prayer, Defiance, Concentration; Vigor 16, 19, ..; Meditation players only), none in town but the caster | the player only | V run (section 2, 3) + S | none today; buildable: game side exists (`palaura_apply.py friendly`), d2d side a `Fight` test with a merc unit under the aura once d2d applies auras to it; oracle = the table of section 2 |
| D2 | Prayer heal and mana | heals every filtered unit that is hurt; pays the cost and sets state 85 only on a pulse where a life rose; no heal below the cost | heals the player only, free, always | V run + S | none; buildable as D1 (heal) and through SKMANA's regen harness (state 85 flag in `apply_regen`) |
| D3 | state 85 on regeneration | base and percent bonus zero, flat stat 26 kept | no such state in the regen | V run (game) + S (SKMANA: `rules::mana_per_frame`) | buildable: `FUN_005806f0` with the state (done here, 8 cases) against a d2d regen test mode with a nomanaregen flag |
| D4 | enemy roll | `min .. max-1` (6,000 pulses: 768 never) | `min .. max` (via `missile_blow`) | V run (game) + S; d2d side proven by SKMISS D1 on `missile_blow` | the SKMISS differential (104,000 cases) covers the function; the aura call site is S. Local candidate `fix/missile-roll-top` changes it too |
| D5 | who an enemy aura hits | the 0xa783 filter: not NPCs, corpses, units out of line of sight, anything in town (the pulse returns before the search); Holy Freeze also not prime evil monsters (damage and slow) | every living monster in the circle | V run (sections 2, 4) + S | none; buildable: game side exists (`palaura_apply.py hit`, `palaura_area.py filter`); d2d side a `Fight` test with a wall, Andariel and a town level |
| D6 | the distance test | integer subtile positions, `d2 <= r2`, only rooms near the caster's | float `hypot * 5 <= r` | V run (game) + S; the float / integer boundary effect I | buildable: place a unit on and next to the circle with fractional positions in a d2d test |
| D7 | pulse floor and phase | `max(perdelay, 5)` frames, frame 1 mod p | `max(perdelay, 25)`, from the selection | V read + S | no effect for any Paladin aura (perdelay 50); would only show for a perdelay under 25 |
| D8 | pierce | the caster's element pierce reduces the target's resist, below 0 too | the same arrays passed | V run (300 of 300) + S read, equal | not a deviation, listed for the record |
| D9 | Might with 0 mana | applies (cost 0) | refused at selection (SKMANA D2 floor) | V run (here) + SKMANA | SKMANA's harness (400 of 12,000 Might cases) |

Not deviations (V run): resist arithmetic `roll * (100 - eff) // 100` (the rest of the strike chain is SKMISS's S); the aura calcs (pal-skill-trace.md: 21,760 cases, 0 differ); the cost formula.

## 8. Bot findings (described, not changed), PR candidates, differentials run

| id | finding | file:line | tag |
|---|---|---|---|
| B1 | The resting Prayer is chosen by the Paladin's own life percent only (`restLifePct 70`, `wantAura` and `restAura`): by the rule it pays mana and holds off mana regeneration only on pulses where someone's life rises: a Paladin at full life (resting for mana) gets normal regeneration (as the bot assumes), but a hurt merc or pet inside the radius keeps Prayer paying and blocks the base regeneration even at full life | internal/combat/aura.go:98-101, internal/combat/resource.go:375-386, internal/combat/classdata/profiles.json:111-113 and 144-146 | V run |
| B2 | With mana below Prayer's per-pulse cost (1.0 at level 1, 4.6 at 20) Prayer heals nothing: a rest that waits for life with an empty mana pool gains life only through natural regeneration; `restAura` does not look at mana | internal/combat/resource.go:375-386 | V run |
| B3 | No aura range or reach is modelled: the Paladin's Might / Concentration reach only the units inside `16 + 2 (L-1)` subtiles and never in town but himself; a merc that lags behind, or the merc's own aura (the merc's skill, a different path, U) is not in the model; Holy Fire's radius is 6 + (L-1) subtiles and needs a straight line (walls stop it), so it is no wall-crossing damage source and does nothing in town | internal/combat/aura.go (no range), internal/merc/policy.go:80-93 | V run (game side) |

**PR candidates: none.** The rule (AGENTS.md): a deviation proven with the existing differential harness (mismatches before > 0, after the fix 0). D4's roll deviation is already proven on the real build by SKMISS D1 and has the local branch `fix/missile-roll-top`; the aura call site rides on it (S) and no existing harness runs `Fight::aura_pulse`. D1 to D3, D5, D6 are missing rules with no d2d test mode today.

**Differentials run (the worker host, game side only, each from `tools/emu/run-on-worker.sh <script> <args>`; Blood Moor or Rogue Encampment game of missiles.py, game seed 3, script rng seed 5):**

| Script | Cases | Result |
|---|---|---|
| `palaura_area.py radius` | 2,032 units, 16 (filter, radius) runs | 2,028 equal the model, 4 outside the near rooms |
| `palaura_area.py filter` | 14 unit kinds x 4 filter words + 13 single bits + direct relation calls | table of section 2 |
| `palaura_area.py los` | 1,200 units, one wall-rich area | 76 rejected by bit 0x200; best collision bit 0x4: 1,183 of 1,200 |
| `palaura_flags.py` | 734 MonStats rows x 29 bits | bits 6 boss, 7 primeevil, 8 npc, 27 noAura exact column matches (0 rows differ) |
| `palaura_filters.py` | every skill with an aurafilter | 18 filter words, section 2 |
| `palaura_apply.py friendly 98 1`, `99 5 <hurt> <mana> [state85]`, `town` | 6 runs | sections 3, 5 |
| `palaura_apply.py regen` | 8 cases | state 85 vs base, percent, flat |
| `palaura_apply.py hit 102 1 6000`, `102 5 100`, `114 5 100`, `118 5 200`, `102 5 300 40` | 6,000 + 100 + 100 + 200 + 300 pulses | section 4 |

## 9. What is real and what is fake in the runs (limits)

Real: the Blood Moor / Rogue Encampment game with real rooms, the real player and monster units from `FUN_00555230`, the real tables, the real area search, filter, callback, dos 65 / 66 / 81, strike chain, regeneration function and calc machine. Hand-made or hooked (not the claim): the merc and pets are ordinary monsters whose AI-struct owner block (+0x28 / +0x2c / +0x30) was set to the owner; the party is a hand-made list in the shape `FUN_00540710` reads (a real party creation was not run); a corpse is a unit whose mode word was set; stats and skill levels on the caster are set through `FUN_00627150` on its stat list and `FUN_00643810` / `FUN_006442a0` (fake skill record and level, as `skill_damage.py`); the hook `missiles.py` puts on `FUN_00554200` was removed (the five bytes restored) and the callback's own hook is removed for the apply runs. Not run: a real merc hire (the merc is a class-271 monster, its life stat is 0 in the run), the real party join, hostile players (`FUN_0055b300`), Sanctuary / Conviction / Redemption / Fist of the Heavens, the Freeze slow's effect on movement, bit 0x20000. The d2d side of every deviation is source reading; no number was compared with the real d2d build here. The per-pulse event chain (`FUN_0056cda0` to the do) is pal-skill-trace.md's V read.

## 10. Dead ends

- First harness runs found nothing hostile and no owner: `missiles.py` hooks `FUN_00554200` (hostile = its own foe), and the owner block's id / type words were swapped (`FUN_0058f0d0` pushes the id from +0x2c and the type from +0x30). Both fixed in the script; the first printouts were wrong for those two reasons.
- `FUN_00639db0` is the state setter (unit, state, on), not a stat setter: an early run put states 8, 9, 12 on the caster by mistake; stats go through `FUN_00627150(list, stat, value, layer)`.
- A fresh unit has seed 0 and the roll is stuck at the minimum: the hit runs set the caster's seed (lo random, hi 666).
- Linking a player into the game hash by hand hung the run (U why; the owner lookup then worked without it, so the unit was in the hash already: I): removed.
- The 0x20000 bit (`FUN_00622c40`) was not decoded; the line-of-sight collision mask is a best fit, not read.

## 11. Drill recipe (live, the maintainer's; lab binary `lab-debug-skills`, docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md; one hidden slot, a throwaway Paladin; nothing here was run)

Verbs by name: `debug points <skill> <n>`, `debug stat 8 <256 x n>` / `debug mana full`, `debug states`, `debug mon set <id> life <n> maxlife <n> res fire|cold|light <v>`, `debug spawn`, `debug cast`, `debug merc hire <type>` / `merc hurt <dmg>` / `merc kill`, `step <n>`, `dump stats`, `dump merc`, `monsters`. A second player (party) has no verb: P3 waits.

- **P1 (D1, who is reached).** `debug merc hire 0`, Might: `debug points 98 5`, select Might (right button); the aura radius is `16 + 2 x 4 = 24` subtiles = 4.8 cells. Put the merc at 4 cells, then at 6 cells (walk the Paladin away): read the merc's damage in `dump merc` if it carries it, else compare the merc's hit damage on a fixed monster with and without Might. Expect game.exe: damagepercent 80 on the merc inside 4.8 cells (Might 5 = 40 + 10 x 4), none outside; d2d: no change on the merc at either distance.
- **P2 (D2, D3, Prayer).** `debug points 99 5`, `debug stat 8 <100 x 256>`, `debug stat 6 <40 x 256>` with max life 80, select Prayer, `step 50` at a time reading `dump stats` (mana raw[8], life raw[6]): expect game.exe life +1536 raw and mana -448 raw (1.75) per 50 frames while hurt, mana regeneration absent (compare with the aura off: 8 raw a frame at 100 max mana); at full life: no mana change, regeneration normal. Merc hurt (`debug merc hurt 20`) and the Paladin full: game.exe still pays 448 raw per pulse and holds regeneration off; d2d pays nothing. With `debug stat 8 256` (1 point, under the cost): no heal on game.exe.
- **P3 (D4, D5, enemy hit).** `debug points 102 5`, `debug mon set <rat id> life 100000 maxlife 100000 res fire 0`, stand within 10 subtiles, `step 50` per pulse for 600 pulses, record the life lost per pulse (raw): game.exe 768 .. 1279 (never 1280), d2d 768 .. 1280 (about 1 pulse in 512 loses exactly 1280). With `res fire 50`: loss = `roll * 50 // 100`. With the fire pierce `debug stat 333 40`: +40 % at res 0.
- **P4 (D5, filter).** Same Holy Fire: put a wall between (a rat behind a wall within the radius): game.exe 0 loss, d2d loss; Holy Freeze (`debug points 114 5`) on Andariel (`debug spawn` her, if the verb takes the class): game.exe 0 loss and no state 44, d2d cold damage; Holy Fire in town on a monster: game.exe 0.
- Expected game.exe numbers above; expected d2d today: P1 none, P2 heal on the player only (free), P3 max 1280 occurs, P4 damage in all three.

## 12. Matrix cell updates (proposed text; `docs/research/pal-skill-trace.md` section 6 and 9 rows, the Paladin matrix file does not exist yet)

| Row | Cell | New value |
|---|---|---|
| pal-skill-trace D5 (aura mana) | tag / differential | V run (palaura): Prayer pays 448 raw at level 5 and sets state 85 only on a pulse where a unit's life rose; below the cost no heal; Might free and works at 0 mana. Regen: state 85 zeroes base and percent, not the flat bonus (`FUN_005806f0`, 8 cases). Differential: buildable (SKMANA harness + a flag), drill P2 |
| pal-skill-trace D6 (who an aura reaches) | rule | V run: filter 0x12003 = caster, party, caster's / ally's pets and merc within the radius, never in town but the caster; drill P1 |
| Auras | reach | V run: 0x12003 / 0xa783 / 0x4a283 decoded (section 2); Meditation 0x12001 players only |
| Holy Fire / Shock | enemy hit | V run: one roll per pulse min .. max-1, resist `roll * (100 - (resist - pierce)) // 100`, NPCs, corpses, no line of sight and town excluded; d2d adds the top value and has no filter |
| Holy Freeze | enemy hit | V run: prime evil monsters skipped entirely (no damage, no state 44) |
| Prayer | heal | V run: every hurt filtered unit, clamped at max life, +6 at level 5 per 50 frames |
| MonStats flags | aura filter | D + V run: bit 6 boss, 7 primeevil, 8 npc, 27 noAura (exact columns on 734 rows) |

## 13. Shared-file lines for the merger

In `openspec/changes/palaura/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG). Decomp-queue row (proposed, number assigned by the merger): `- NN Paladin auras apply (PALAURA, 2026-11-02): area search = integer subtile disc, d2 <= r2, caster's room and near rooms; filter 0x12003 passes the caster, party, pets and merc, 0xa783 living attackable monsters in line of sight not in town, Holy Freeze (0x4a283) skips prime evil; enemy roll min .. max-1, resist roll x (100 - (resist - pierce)) / 100; Prayer heals every hurt filtered unit and pays mana / sets state 85 only when a life rose (d2d: player only, free, no filter, roll top, no LOS or town rule); no PR candidate (docs/research/palaura-trace.md).` Ledger: no row changed (game side only, no d2d differential). Glossary: no new term.
