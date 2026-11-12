# fid-pal-attack (TASK-318, matrix rows SK04 SK16 and the Paladin skill rows): Paladin Attack and Zeal speed, mana, and the profile bands

> status: finished 2026-11-11 by agent decomp9 (Sonnet), offline. Result: **swing speed and mana for Attack, Smite and the first Zeal strike are known (V run by earlier traces, D tables here); the Zeal strike interval (the rollback) is NOT resolved (U, dead end below); the bot's Paladin bands equal the skill requirement levels (D).** No PR candidate. Consumer: TASK-142 (paladinsmoke).

Tags: V run, V read, D (1.14d tables: Skills.txt, weapons.txt, MonStats via `tools/emu/mpq-table.sh`; AnimData.d2 via `tools/emu/d9_animdata.sh`, parsed here), S (existing proof), I, U.

## Q1. Plain Attack: frames, speed, ticks (S for the rate, D for the frames, I for the class mapping)

- Rate rule: S, V run in [skiasgame2-trace.md](../../research/skiasgame2-trace.md): game.exe's attack branch of FUN_00623f50 equals d2d's `attack_speed`, **198,988 cases, 0 mismatch**; ticks = `ceil(256 x frames / rate)`, 81 cases 0 mismatch. With IAS 0: rate = AnimData speed x clamp(100 - WSM, 15, 175) / 100; with IAS i: add eff(i) = 120 i / (120 + i).
- Paladin frames (D, AnimData.d2 records of the same MPQ, speed 256 in all of them; the frame carrying the action flag in brackets): attack with one-hand sword class `PAA11HS` **15 (action frame 7)**, one-hand thrust `PAA11HT` 17 (8), two-hand sword `PAA12HS` 18 (8), two-hand thrust `PAA12HT` 20 (8), staff `PAA1STF` 18 (9), bare hands `PAA1HTH` 14 (7), bow 16 (8), crossbow 20 (10); skill mode S1 one-hand sword `PAS11HS` 12 (7) (Smite's mode, Skills anim S1). Which record a given weapon uses follows its `wclass` column (I: the name is class code + mode + weapon class; not run).
- Ticks per swing (D arithmetic on the S rule; WSM = weapons.txt `speed`; the columns are IAS 0, 20, 40, 60). The game runs 25 ticks a second.

| Weapon (1hs class unless noted) | WSM | min-max damage | ticks at IAS 0 / 20 / 40 / 60 |
|---|---:|---|---|
| Club | -10 | 1-6 | 14 / 12 / 11 / 10 |
| Hand Axe, Short Sword, Mace, Broad Sword, Crystal Sword, War Sword | 0 | 3-6, 2-7, 3-10, 7-14, 5-15, 8-20 | 15 / 13 / 12 / 11 |
| Scimitar | -20 | 2-6 | 13 / 11 / 10 / 10 |
| Flail | -10 | 1-24 | 14 / 12 / 11 / 10 |
| Axe, Morning Star, Bastard Sword | +10 | 4-11, 7-16, 7-19 | 17 / 15 / 13 / 12 |
| War Hammer, Falchion | +20 | 19-29, 9-17 | 19 / 16 / 14 / 13 |
| Short Spear (1ht, 17 frames) | +10 | 2-13 | 19 / 16 / 15 / 14 |
| Javelin (1ht) | -10 | 1-5 | 16 / 14 / 13 / 12 |

So a Paladin at 0 IAS swings a sword or mace 1.67 times a second (15 ticks), a Scimitar 1.92 (13). Whether the weapon's WSM reaches the unit as stat 68 = 100 - WSM is the item side that skiasgame2 did not read (I there). The first-strike time is ceil(256 x (action frame + 1) / rate) ticks after the swing starts (I: the strike is the action event of the flagged frame; the event scheduler FUN_005539b0 is traced in fid-hit-recovery.md).

## Q2. Mana (V run through SKMANA, D for the parameters)

SKMANA ([sk-mana-trace.md](../../research/sk-mana-trace.md), 130,840 cases, V run): cost = `(mana + lvlmana x (lvl - 1)) << manashift` in 256ths, the payment never below minmana x 256, and the check has no floor.

| Skill | req clvl | mana, lvlmana, shift | cost at skill level 1 / 5 / 10 / 20 | when paid |
|---|---:|---|---|---|
| Attack | 1 | 0, 0, 8 | 0 | n/a |
| Sacrifice | 1 | 0, 0, 8 | 0 (and 8 percent of the damage dealt back to you, capped at the target's life: pal-smite-zeal-sacrifice-trace.md V read) | n/a |
| Smite | 1 | 2, 0, 8 | 2 | at the strike |
| Zeal | 12 | 2, 0, 8 | **2 per cast** (the whole 2 to 5 strike sequence) | once at the start (srvstfunc 37, V read in pal-smite-zeal-sacrifice-trace.md) |
| Charge | 12 | 9, 0, 8 | 9 | start |
| Vengeance | 18 | 16, 1, 6 | 4.0 / 5.0 / 6.25 / 8.75 | start |
| Blessed Hammer | 18 | 20, 1, 6 | 5.0 / 6.0 / 7.25 / 9.75 | at the action frame (no start function; pal-skill-trace.md) |
| Holy Shield | 24 | 35, 0, 8 | 35 | start |
| Prayer (aura, switch cost) | 1 | 16, 3, 4 | 1.0 / 1.75 / 2.69 / 4.56 | at the switch (U that an aura pays at all: pal-aura-trace.md) |

(Costs are in mana points; a value below 1 is paid in 256ths. minmana 1 raises the payment only where the formula gives less than 1.)

## Q3. Zeal: the strike interval (U, dead end)

- Known (S, V read): Zeal is `anim A1` (so its mode is the plain attack mode, not the sequence mode 18: the -30 points of skiasgame2 D1 do not apply, I), 2 / 3 / 4 / 5 strikes at skill level 1 / 2 / 3 / 4+, Param2 = 100 (rollback percent), each strike except the last calls FUN_0056e210 -> FUN_00553b10 (game, unit, 100).
- d2d waits the full swing between strikes (`strike.cpp:901`, `pmode_until`, S: pal-smite-zeal-sacrifice-trace.md D9): a level-4 Zeal on a Short Sword = 5 x 15 = **75 ticks (3.0 s)**.
- game.exe: FUN_00553b10 removes the unit's events of type 0 (flagged frames) and 1 (the end of the mode), moves the position field (+0x44) by the rollback and reschedules the events of the frames still ahead. A first native run of it (a hand-made unit with the PAA11HS record: 15 frames, action flag on frame 7, rate 256..448, game frame 1000) printed events at game frame + 7 (flag 1) and +15 (end) at rate 256, +4 and +9 at rate 448, with the position field set to the game frame number x 256. That means the field is a time reference, not an animation frame, and **I could not resolve its meaning in this job**: the harness answers were therefore not used, the script was deleted, and the result is **U**. The expected value is somewhere between "one tick" and "the full swing"; the community figure of about 9 ticks a strike at 0 IAS for a sword is S and was not checked here.
- Differential that would settle it: run FUN_005dbc60 (the Zeal do) natively on a real player unit created in a booted game (`mm_launch.G` makes monsters; a player unit needs the player creation path, FUN_00555230 with unit type 0 and an inventory), pump the event queue of FUN_005417d0 (the same hook as `hit_len.py`) until the last strike, record the game frames of each do. Oracle = the event frames; d2d side = `Fight::strike` Zeal with `pmode_until` read per strike (test_game scenario with a dummy target). Cost: about one day of harness work (U).

## Q4. The bot's Paladin bands against the rules (D)

`internal/combat/classdata/profiles.json` (paladin): band `minLevel 12 requires Zeal` -> attack order Zeal, Attack; band `minLevel 18 requires Blessed Hammer` -> Blessed Hammer, Zeal, Attack; aura list Prayer (rest life 70), Concentration, Holy Fire, Might, Thorns.

- Zeal's required level is 12 and Blessed Hammer's 18 (Skills.txt `reqlevel`, D): the bands equal the earliest level a skill can be learned. A band needs the skill actually learned (`requires`), so a character that skipped Zeal falls through to Attack (right).
- Auras: Prayer 1, Might 1, Holy Fire 6, Thorns 6, Concentration 18 (all D); in the clvl-12 band the list is the same, an aura the character has not learned is skipped (`requires` semantics not re-read: U).
- Mana: at Zeal's 2 per cast and Attack's 0, a Paladin at the level-12 band needs no regeneration budget beyond a few potions; Blessed Hammer is the first skill that matters (5 to 10 per cast at skill level 1 to 10 from clvl 18). The mana pool of a fresh Paladin and regeneration are ST01 / ST04 rows (V live / V read).
- Smite is not in any band: `Smite` needs a shield (V read) and costs 2 mana for a stun; not a defect.

## Deviations

- D1 (U, from pal-smite-zeal-sacrifice-trace.md D9): Zeal strike spacing. Game: unresolved, at most the swing length; d2d: the full swing per strike. If the game is faster than the swing, d2d's Paladin kills slower than the real one with Zeal and the bot's band thresholds measured on d2d are pessimistic. Differential: buildable (Q3 recipe), not built here.
- D2 (carried): sequence skills (Charge) lose 30 points of attack rate in the game, none in d2d (skiasgame2 D1, V run on the game side, not against d2d's code); Charge is in no band.
- No new d2d deviation is proven in this card. **No PR candidate.**

## Bot findings (described, not changed)

- B1 profiles.json paladin bands: see Q4. No change needed.
- B2 `internal/combat/necro.go:42` `buffStates` lacks Holy Shield (covpal-trace.md B2, recast by clock only): carried, not re-checked.
- B3 any Zeal-rate assumption in the bot's kill-time model (grep found none in `internal/combat/classdata`; a timing model would read it from the profile) should be marked U until Q3 is settled.

## Drill recipe (fork verbs; written, not run)

1. Attack ticks: `debug equip` a Short Sword, `debug spawn` a training dummy, `cmd attack` a unit, `step 1` per tick and print the Paladin's mode and the dummy's life: expect the strike at tick 8 (frame 7 + 1) and the next swing at tick 15 (d2d, game.exe expected equal by Q1).
2. Zeal: `debug points 106 4`, `cmd cast 106 unit`, per tick print the dummy's life and count the ticks between lowering events. The lab binary is d2d, so this measures d2d only (expected 15 apart); the game.exe side needs the Q3 differential.
3. Mana: `debug points 106 1`, cast Zeal five times, read mana before and after (`dump stats`): expect 10 less (2 each, once per cast).

## Matrix-cell updates (proposed text)

- SK04: append "UPDATE 2026-11-11 (decomp9, fid-pal-attack.md): Paladin skill costs tabulated from the SKMANA formula (Zeal 2 flat per cast; Blessed Hammer 5.0 + 0.25 per level)."
- SK16 / Paladin skill rows: "Zeal strike spacing U; d2d waits the full swing; a native run of the rollback function FUN_00553b10 was attempted and not resolved (fid-pal-attack.md Q3)."

## Limits and dead ends

- AnimData was parsed by hand here (the 8-byte code, frames per direction, speed word, action flag array): the 3,558 records equal the file size exactly; the weapon-to-record mapping is by name (I).
- FUN_00553b10 harness: unresolved (Q3); the script was removed so a half-right oracle does not survive.
