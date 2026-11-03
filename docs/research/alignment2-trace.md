# Alignment (stat 0xac) and the undead-blow gate: game.exe 1.14d vs d2d vs our bot (agent alignment2, 2026-11-02)

> status: research, current (card alignment2; the HIGH finding of docs/research/pr-skmelee3-verifier.md). Offline: no game, no d2d slot; nothing pushed, posted or filed. Disassembly of the read-only game.exe copy on the worker host, run through upstream's emulator (unicorn) with `tools/emu/al2_run.py`. Tags: V read = read in the disassembly, V run = game.exe's own code run under the emulator and counted, D = data (the MPQ stack's MonStats.txt), S = an existing doc, I = inference, U = unverified. d2d was NOT read here: the PR candidate (local/skmelee3-v1) is on a fork branch outside this sandbox; its rule is taken from the verifier's text (S) and no d2d code line is cited.

## Behaviour (the answer first)

1. **The gate.** `FUN_0057b7d0` (damage build) runs the stat 120 / stat 121 / undead (`FUN_0057b660`) / stat 180 (`FUN_0057b6a0`) block when the defender is a monster (type 1) and the attacker is **type 0 (player) or `FUN_006259b0(attacker) == 2`** (V read 0x57b805..0x57b82a; V run B: 288 cases, 0 differ). The attacker's own type 0 skips the alignment call, so a player gets the block whatever its stat 0xac is.
2. **FUN_006259b0 = "alignment".** V read 0x6259b0..0x6259ee: no unit (0) -> 0; `[unit + 0x5c]` (stat list) null -> 0; unit type above 1 -> **2** (only with a stat list: the verifier's note "any type above 1 returns 2" misses the null-list exit first); type 0 / 1 -> stat 0xac read from the stat list of **state 0x69** (`FUN_006256b0` -> `FUN_00625650`: walks the chains at +0x3c and +0x40 of the list for the node whose +0x14 is the state), 0 when the unit has no such list. V run A: 168 cases, 0 differ.
3. **Who writes it.** One setter, `FUN_005543b0(unit in ECX, alignment in DL, flag)`: builds the state-0x69 list and writes stat 0xac (0x554467..0x554480); V run D: setter then getter, real code, unit types 0 / 1 x classes x alignment 0 / 1 / 2 = 18 round trips, 0 differ. 21 call sites (V read, list in the table).
4. **Values by kind.** Every unit that belongs to the player's side gets **2**; hostile monsters 0. Table below. So in game.exe the undead bonus applies to the blows of the player, hired mercenaries, Shadow Warrior / Master, Raise Skeleton / Skeletal Mage, golems, Revive, Valkyrie, Hydra, druid summons, sentries and converted monsters, **provided the stats it reads (the attacker's stat 122, a Blunt weapon in `FUN_00535bc0`) exist on that unit**. Hostile monsters (alignment 0) and the two Align-2 rows (cow, camel: alignment 1) do not get the block.

## Q1..Q6

| # | Question | Answer | Addresses | Tag |
|---|---|---|---|---|
| Q1 | What does `FUN_006259b0` return for a player? | Type 0: stat 0xac of the state-0x69 list; the player is created with 2 (`FUN_00555230` jump table 0x5554e8[0] = 0x55536f -> `FUN_005348c0` -> `FUN_005543b0(unit, 2, 1)` at 0x53495a). Irrelevant to the gate: type 0 passes it by type | 0x6259b0, 0x555368..0x555374, 0x534954..0x53495a | V read; getter V run A, setter V run D |
| Q2 | (a) hired mercenary | **2.** The five hireling classes (roguehire 271, act2hire 338, act3hire 359, act5hire1 560, act5hire2 561) have MonStats Align 1; the spawn code maps the record byte `+0x4c`: 1 -> 2 (and `|= 0x20000` on the unit flags), 2 -> 1, else 0. Re-attach (`FUN_00573270`, which sets 2 and `+0xc4 |= 0x4020200` for any unit it links to a player) agrees | 0x5b2e94..0x5b2ec2 (record = table 0xa78, stride 0x1a8), 0x573270..0x5732b3 | V read (mapping); the `+0x4c` byte V run C (734 rows, 0 differ from the Align column); that `FUN_00573270` also takes the merc: I (S: pets-merc-trace.md lists it with the summons and the merc) |
| Q3 | (b) Shadow Warrior / Shadow Master; Revive; Raise Skeleton / Skeletal Mage | Shadow Warrior 417, Shadow Master 418, necroskeleton 363, necromage 364, golems 289..292: MonStats Align 1 -> **2** at spawn. **Revive** (do 58 = `FUN_005c56c0` -> `FUN_005c55c0`) sets **2** explicitly on the revived monster at 0x5c564a..0x5c564c, whatever its class (a hostile class has Align 0, so this call is what flips it) | 0x5c564c, table 0x7322b0[58] | V read; the class bytes V run C; Revive's own run: not done (U for the revived unit's later stats) |
| Q4 | (c) other player-owned pets | Valkyrie 357, Hydra hydra1 351 (the do-144 summon `FUN_005ca910` also copies the caster's alignment when `FUN_0044be50(caster) == 1`, 0x5caafa..0x5cab05: I, a player caster fails that test and the spawn value 2 stands), druid pets (wolf 420, fenris 421, bear 428, hawk 419, vines 425 / 427, totems 422..424, cycle 426), Dopplezon 356, sentries (bladecreeper 413, charged bolt 411, wake 410, inferno 415, death 416; do 44 / 45 call `FUN_005d5e10`, which sets 2 at 0x5d5feb), bonewall 344: all Align 1 -> **2**. Conversion (do 79 = `FUN_005d0350`) sets 2 on the converted monster (0x5d04ef) and its undo `FUN_005d01a0` sets 0. Lightning Spire 371: Align blank -> 0 | 0x5d5feb, 0x5d04ef, 0x5d01b9 | V read; classes V run C |
| Q5 | (d) the player's own unit type | Type 0: block runs, no alignment read (see Behaviour 1) | 0x57b805..0x57b81f | V read; V run B (type 0 attacker, every stat-list shape, 0 differ) |
| Q6 | Does the undead bonus then apply to those units' blows? | The **gate** says yes for every alignment-2 unit (V run B). The **bonus size** depends on what the unit carries: `FUN_0057b660` adds (+50 when `FUN_00535bc0(attacker)` is a Blunt, ItemTypes 57) + attacker stat 122 to the record's ED when the sum is over 0 and the target is undead. A monster with no inventory (`[unit + 0x60]` null: every pet) has no weapon, so 0 + its own stat 122 (normally 0): no visible effect. A merc has an inventory, so a Blunt weapon or stat 122 on merc gear counts (the function takes the weapon from the inventory for any unit with one; that it returns the merc's hand weapon: I, 0x535bc0..0x535bf9 read for a monster not completed) | 0x57b660..0x57b69d, 0x535bc0 | V read; V run (`FUN_0057b660` itself: 60 / 60, sk-melee-trace.md Q6, S) |

Alignment per kind (the table the brief asked for):

| unit kind | how it gets alignment | value (`FUN_006259b0`) | undead block for its blow |
|---|---|---|---|
| player (type 0) | `FUN_005348c0` at creation | 2 | yes (by type) |
| hired merc (5 classes) | spawn map Align 1 -> 2; attach `FUN_00573270` | 2 | yes |
| Shadow Warrior / Master | spawn map | 2 | yes |
| Raise Skeleton / Skeletal Mage | spawn map | 2 | yes |
| golems (clay / blood / iron / fire) | spawn map | 2 | yes |
| revived monster | `FUN_005c55c0` sets 2 | 2 | yes |
| Valkyrie, Dopplezon | spawn map | 2 | yes |
| Hydra | spawn map (+ copy of the caster's, I) | 2 | yes |
| druid summons, ravens, vines, totems | spawn map | 2 | yes |
| assassin sentries, Blade Sentinel | spawn map; `FUN_005d5e10` | 2 | yes |
| converted monster (Conversion) | `FUN_005d0350` | 2 | yes |
| Attract / confuse target | `FUN_005c3b90` and callbacks set 1, undo 0 | 1 / 0 | no |
| ordinary hostile monster | spawn map (619 of 734 rows) | 0 | no |
| cow, camel (Align 2) | spawn map | 1 | no |
| any unit of type 2..5 with a stat list | `FUN_006259b0` type > 1 exit | 2 | yes (is such a unit ever the attacker of this function? U) |
| any unit with no stat list | null `+0x5c` | 0 | no |

## Runs (V run, counted; `tools/emu/al2_run.py`, output kept in the card)

| part | what | cases | differ |
|---|---|---|---|
| C | game.exe's loaded MonStats record byte `+0x4c` vs the Align column | 734 rows | 0 |
| C | spawn alignment over all rows by the 0x5b2e94 map | 113 rows alignment 2, 2 rows alignment 1, 619 rows alignment 0 | (read map, not a comparison) |
| D | real `FUN_005543b0` then real `FUN_006259b0` (a pool allocator stub, `FUN_00626d40` for the unit's own list) | 18 | 0 |
| A | real `FUN_006259b0`, only the raw stat read `FUN_00625420` answers from a table: unit type 0..5 x 7 stat-list shapes x stat 0..3 | 168 | 0 |
| B | real `FUN_0057b7d0` up to 0x57b882 (stat getters, `FUN_00535d10` hooked, `FUN_0057b660` counts entry): attacker type 0..5 x stat-list shape x alignment 0..3 x defender type 0..2 | 288 | 0 |

Limits of the runs: A and B check the control flow against my reading of the code, not against d2d; D's `FUN_005543b0` ran with a pool stub and without the `FUN_00554340` follow-up side effects of a real game; the spawn function `FUN_005b2a00` itself and the summon functions were not run (V read only), so "what value a live merc or Shadow carries" is V read + V run of the pieces, not a live stat read (drill below).

## Deviations d2d vs game.exe (D1..)

| # | game.exe | d2d (S: the verifier's text, not re-read) | tag | differential |
|---|---|---|---|---|
| D1 | merc and Shadow blows take the undead / demon / stat 120 / stat 180 block (alignment 2) | the candidate's `not_player()` (combat.hpp, merc.cpp, pets.cpp, 3 call sites) excludes mercs and Shadows | game.exe side V read + V run B; d2d side S | Buildable: feed the same attacker (a merc with a Blunt weapon / stat 122, a Shadow with stat 122) to d2d's merc / pet blow and to `FUN_0057b7d0` run as in part B, compare the record's ED (+0xc). Not run: the candidate is not in this sandbox |
| D2 | the same for every pet and summon (skeleton, golem, Valkyrie, druid summon, sentry): the gate passes, the bonus is whatever the unit's own stat 122 and (no) weapon give | d2d: `not_player` excludes them; their stat 122 is normally 0 so the numbers agree except for units given stat 122 | game.exe V read + V run; d2d S | same harness, pets as attacker |
| D3 | a hostile monster is alignment 0 and never gets the block | d2d (S: verifier says it models `align == 2` for the player's side) | I | none needed |

No PR candidate: nothing here was proven against d2d's own code by the existing differential harness (no before/after run); D1 / D2 feed the open verifier finding of pr-skmelee3-verifier.md instead.

## The exact rule d2d should implement (for the candidate's D2)

- Drop `not_player` for the **merc and the Shadow Warrior / Master** (and for every summon): the undead block applies to any attacker that is the player or has alignment 2, against a monster. Alignment 2 holds for every player-side unit in the table above (V read for the setters, V run for the Align byte of all 734 rows).
- The block uses the **attacker's own** stat 122 (+ the Blunt +50 only if the unit holds a Blunt weapon). For pets without an inventory that is stat 122 alone, so d2d may keep a pet at its own stat 122 (usually 0); this is not a behaviour change for the Sorceress.
- Cite `FUN_006259b0` and `FUN_005543b0`/`FUN_005b2a00` in the comment; the comments "FUN_0057b660 is for a player unit only" and `not_player` state the wrong rule (as the verifier said).
- Hostile monsters (alignment 0), neutral (1) are excluded.

## Bot findings (B1.., described, not changed)

- **B1.** The bot's melee model is the player's blow only (`internal/combat/melee.go:62` Blunt from the wielded weapon; `internal/resource/melee.go:7` `BluntUndeadPct = 50`, :89 the term; Foe undead from MonStats LUndead / HUndead at `internal/combat/melee.go:86`). It does not model a merc's blows at all, so the alignment rule above has no counterpart and nothing to fix; any later merc damage model must use "alignment 2" not "player only".
- **B2.** `BluntUndeadPct`'s comment cites `FUN_0057b660` as "V run, trace D2": the formula there is the ED add; the gate in front of it is `FUN_0057b7d0` (player or alignment 2). Doc-only note; the player case is right either way.

## Drill recipe (docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md; run later, not here)

1. In the lab binary hire a merc (any act), then read stat 0xac of the merc's stat list (state 0x69) with the fork's stat-read verb on the merc unit: expect 2 (Q2). Same for a Shadow Warrior (assassin), a Raise Skeleton, a Valkyrie, a Hydra, a revived monster, a converted monster: expect 2 each; an ordinary Fallen: 0; a town NPC (akara): 2.
2. Equip the merc with a stat-122 item (undead damage) and hit a Skeleton; read the damage-record ED (+0xc) before the roll with the damage breakpoint verb: expect +stat 122 percent. Repeat with the item removed: 0.
3. Revive: kill a hostile Zombie, Revive it, read its alignment before and after: 0 then 2.

## Dead ends

- `FUN_005dfee0` / `FUN_005dfbf0` look like state-0x69 users but are an unrelated event path (they only call `FUN_006259b0` to filter targets); not followed.
- Reading `FUN_00573270` as "the merc": its caller chain (0x5770e0 / 0x5774f0 / 0x579180) was not named; kept as I.
- The first native try of the setter (D) returned 0 for alignment 1 / 2: my fake unit lacked its own stat list (the real unit gets it from `FUN_005348c0`'s call to `FUN_00626d40`); that was a harness fault, fixed, not a game finding.
- No ledger row was added: the runs compare game.exe against its own reading, not against d2d (docs/coverage/README.md allows TESTED-* only against d2d's real rules), and READ-ONLY does not fit a single existing subsystem row.

## Matrix-cell updates (proposed text)

- `docs/research/sk-melee-trace.md` / the melee matrix row "stat 122 / Blunt vs undead": append "gate = player or alignment 2 (FUN_0057b7d0, alignment2-trace.md): mercs, Shadows and all summons pass it".
- merc rows of `docs/research/pets-merc-trace.md`: add "alignment 2 (MonStats Align 1 -> 2 at spawn; alignment2-trace.md Q2)".

## Glossary

Missing term: "alignment (stat 0xac)" (0 hostile, 1 neutral, 2 the player's side) for GLOSSARY.md.

## Shared-file lines for the merger

See `openspec/changes/alignment2/merge-lines.txt`.
