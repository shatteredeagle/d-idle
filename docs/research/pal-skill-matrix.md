# Paladin skill matrix: what each skill adds over plain Attack, ranked by bot impact (PALSKILL, decomp13)

> status: research, 2026-11-12, agent decomp13 (Sonnet), offline (emulator and a d2d test binary on the worker host; no game, no live d2d, no Go change, nothing pushed or posted). The Sorceress model is [sorc-skill-matrix.md](sorc-skill-matrix.md). Companion trace: [pal-zeal-trace.md](pal-zeal-trace.md). Builds on [pal-smite-zeal-sacrifice-trace.md](pal-smite-zeal-sacrifice-trace.md), [palskill-trace.md](palskill-trace.md), [pal-skill-trace.md](pal-skill-trace.md), [palaura-trace.md](palaura-trace.md), [pal-aura-trace.md](pal-aura-trace.md), [fid-pal-attack.md](../probes/fid/fid-pal-attack.md), [fid-zeal.md](../probes/fid/fid-zeal.md), [awesome-diablo-mining-2-2026-11-12.md](awesome-diablo-mining-2-2026-11-12.md) (the leveling proposals). The mined helper repositories were read for names only; nothing was copied.

Tags: **V run** = executed (game.exe function under the emulator, or d2d's real classes); **V read** = read in the disassembly; **D** = 1.14d patched tables (`tools/emu/mpq-table.sh`); **S** = an earlier trace of ours / upstream source; **I** = inferred (this card's arithmetic model `tools/emu/pm_dps.py` is I throughout: it multiplies V / D / S inputs, it does not run game.exe); **U** = unverified. Every number called V run below is quoted from a run of game.exe's code; the rest says what it is.

## Answer

1. **Biggest lever the bot does not use: Sacrifice.** Level 1 gives +180 percent damage and +20 percent attack rating on the same 15-tick swing for no mana (D, V read; calcs V run 0 differ, palskill-trace.md), at a life price of 8 percent of the damage dealt (capped at the target's life, V read). The model (I) puts a clvl-6 Paladin at **25 damage per second against plain Attack's 13** (defence 15; 18.5 against 9.2 at defence 60) for about 2.0 life per second. The v2 table buys the point at clvl 3 and the combat profile never casts it (B1). The life price is U as a net cost (Q6).
2. **Zeal is worth about 1.74 times the strikes of an Attack in game.exe and about 1.0 times in d2d.** The next strike lands 7 ticks after the last in game.exe (rollback of Param2 100, V run) and 15 in d2d (V run through its real Fight); a 5-strike cast is 43 against 75 ticks at 0 IAS (pal-zeal-trace.md). Everything the bot measured about Zeal is therefore a d2d number. Zeal at skill level 1 (the v2 table's clvl 12) is 2 hits, damage-wise below a one-point Sacrifice: the model gives 23.8 dps (game) against Sacrifice 28.7 and Attack 16.5; Zeal 4 (clvl 15 in v2) gives 31.8, passes the one-point Sacrifice by 10 percent and pays no life.
3. **Might is the right plain-fight aura; Holy Fire is not, below about 3 enemies in reach.** Might adds +40 percent (+10 per level) damage on every swing (D, ln34): Might 4 turns 8.6 dps into 13.1 at clvl 6 and Might 8 turns 9.4 into 16.5 at clvl 12 (I). Holy Fire level 4 deals 2.5 to 4.5 points once per 50 frames to each monster within 9 subtiles: **1.75 damage per second per monster** (V run: pm_dmg.py, FUN_00644d50 / e40; pulse period D + V read). The bot's aura ladder takes the first learned entry, and Holy Fire sits before Might (B2); the v2 table learns Holy Fire at clvl 16 to 19, after which Might would no longer be on in fights.
4. **Smite is the weakest damage source and the only sure hit.** It deals the shield's damage (Buckler 1 to 3 points), +15 percent +15 per level, +1 percent per Strength: about 11 dps at clvl 12 against Attack's 16.5 (I), but never misses or gets blocked in game.exe (V read; d2d lets block cancel it, D8 of pal-smite-zeal-sacrifice-trace.md) and stuns 15 + 5 per level frames and knocks back. Right as a boss opener, wrong as the main attack: the bot's current ladder leaves it out (good).
5. **Best order, damage only (I, section 5):** Sacrifice 1 at clvl 2, Might 1 at clvl 3, then Sacrifice (each point +15 percent ED) until Zeal at 12; Zeal 4 is worth more than a fifth Sacrifice point only when the life price matters. With a life budget (U) the order is Might and one Sacrifice first, Zeal 4 at clvl 12 to 15, later points to Sacrifice only as potion cost allows.

## 1. Sources read and what ran

Read: AGENTS.md, rules-short, decomp-queue.md, sorc-skill-matrix.md (template), the Paladin traces above, fid-zeal.md, fid-pal-attack.md, internal/combat/classdata/profiles.json, internal/combat/aura.go, internal/plandata/build-paladin-normal30-v2.json. Ran on the worker host (nice): `zd_zeal_test` (d2d Fight, 12 runs), `zd_game.py` (FUN_00553b10, 6 rates), `pm_dmg.py` (FUN_00644d50 / e40 via the pal_dmg oracle, Holy Bolt and Holy Fire levels 1 to 8), `mpq-table.sh skills / charstats / weapons / monstats` for the D rows. Not run: any live game, any d2d slot.

## 2. Common facts the rows refer to

- Rates: ticks per swing = ceil(256 x frames / rate), rate 256 at 0 IAS and 0 WSM (S, skiasgame2-trace.md, 198,988 cases). Starting Short Sword (D): PAA11HS 15 frames, action frame 7; Smite's mode S1 (PAS11HS) 12 frames, action 7; Paladin cast PASC1HS 16 frames (D, AnimData; the action marker of a cast record is a 2 at frame 9, meaning U). 25 ticks a second.
- To-hit: AR = (Dex - 7) x 5 + CharStats ToHitFactor 20 (D: 20); chance = AR' / (AR' + DEF) x 2 alvl / (alvl + dlvl), 5 to 95 percent, AR' = AR x (100 + skill bonus) / 100 (S, sk-melee-trace.md). Paladin start (D): Str 25, Dex 20, Vit 25, Ene 15, LifePerLevel 8 and ManaPerLevel 6 (in fourths). Start kit Short Sword 2-7 (Str bonus 100, speed 0) and Buckler.
- Damage: weapon average x (100 + Str x StrBonus/100 + skill ED + Might) / 100; whether Might's stat adds to the same sum is I (the trace reads Might as `damagepercent` = stat 25 = the off-weapon ED term).
- Auras are the selected right skill (S); only one at a time (I). Pulse: a scheduled event every 50 frames (2.0 s), aligned to game frame 1 mod 50 (V run, pal-skill-trace.md); the self state lasts to the next boundary + 2 frames. Range (D, ln12, subtiles): Might / Prayer / Thorns / Defiance / Blessed Aim / Concentration 16 + 2 per level, Vigor 16 + 3 per level, Holy Fire / Shock / Freeze 6 + 1 per level (5 subtiles are one cell).
- Mana: cost = (mana + lvlmana (lvl - 1)) << manashift in 256ths (V run, 130,840 cases, sk-mana-trace.md).

## 3. Per-skill matrix (ranked by bot impact)

Columns: (a) cast / swing and ticks, (b) mana and when, (c) hit and damage rule, (d) first-point value, (e) d2d against game.exe, (f) bot use.

### 3.1 Sacrifice (96, reqlvl 1, A1; srvstfunc 29, srvdofunc 64) rank 1 (largest unused gain)
- (a) A1, the plain attack animation: 15 ticks with the Short Sword at 0 IAS; UseAttackRate 1 (D). V read FUN_005ce790 / FUN_005ce8e0.
- (b) mana 0 (D, V run). Life: the do takes 8 percent (calc2, Param3 8) of the physical damage dealt, capped at the target's current life, as direct damage flag 0x1000; it can kill the caster (V read; pal-smite-zeal-sacrifice-trace.md; which life value, before or after the blow, U).
- (c) ED = 180 + 15 (lvl - 1) + 15 Redemption + 5 Fanaticism (calc1, V run 0 differ); AR +20 + 7 (lvl - 1) percent (V run); standard block / dodge / hit roll (V read).
- (d) clvl 2 (27 Str, Short Sword): Attack 8.1 dps, Might 1 + Attack 10.6, **Sacrifice 1: 20.1 dps** (I). clvl 6: 13.1 (Might 4) against 25.3 at 2.0 life/s.
- (e) d2d builds all three melee skills (S, skill_built); price rule equal in kind; no known deviation beyond the shared ones. Differential: the calc slots are V run (palskill-trace.md, 40,720 cases); the price is read only: buildable through pal2_test + a Fight price run (drill C1).
- (f) never cast (profiles.json Paladin bands: [Zeal, Attack] from clvl 12 requires Zeal, [Attack] below). B1.

### 3.2 Zeal (106, reqlvl 12, needs Sacrifice 1; A1; srvstfunc 37, srvdofunc 13) rank 2
- (a) A1; 2 / 3 / 4 / 5 strikes at skill level 1 / 2 / 3 / 4+ (calc1 = min(2 + lvl - 1, 5), V run); first strike at the action frame (7 ticks, sword class); **game.exe: the next strike `action` ticks later, 7 at 0 IAS, 6 at IAS 20 to 40, 5 at IAS 40 with WSM -20 (V run); a 5-strike cast 43 ticks. d2d: the full swing, 15 / 13 / 12 / 10 per strike, a cast 75 (V run through the real Fight).**
- (b) 2 mana, once per cast (srvstfunc 37), V run through the d2d side (3 casts, 6 mana) and V read in game.exe (fid-pal-attack.md). No `delay` / `perdelay` (D).
- (c) ED = (lvl < 5 ? 0 : (lvl - 4) x 6) + 12 per Sacrifice level (calc2, V run); AR +10 + 10 (lvl - 1) percent (V run); standard hit roll per strike; a dead or out-of-reach target retargets by the next-higher unit id (V read, 0x20003 filter).
- (d) clvl 12 (v2: Zeal 1, 2 strikes): 23.8 dps (game) / 17.5 (d2d) against Attack 16.5 and Sacrifice 1 28.7; Zeal 4 (clvl 15): 31.8 / 18.2 against 16.7 and 29.0 (I, ssd, defence 15; with a Mace 3-10: 45.3 / 25.9 against 23.8). Below skill level 4 the extra strikes are the only gain (no ED).
- (e) D1 spacing (above), no PR candidate (pal-zeal-trace.md); D2 an idle tick between held swings (d2d).
- (f) in the bands from clvl 12 (requires Zeal). B3.

### 3.3 Might (98, reqlvl 1, aura) rank 3
- (a) none. Pulse 50 frames; range 16 + 2 per level subtiles; reaches the caster, party, merc and pets (filter 0x12003, V run palaura-trace.md). Mana 0 (V run).
- (c) damagepercent = 40 + 10 (lvl - 1) (D, ln34 = Param3 + Param4 per level).
- (d) +32 percent at Str 25 for one point (I: 100 + 25 -> 100 + 25 + 40); with 4 points +52 percent of the Attack damage: 13.1 against 8.6 dps at clvl 6; the fifth point adds +0.6 dps (13.8) (I).
- (e) d2d friendly auras: the caster's own stats only, no filter, no merc (palaura-trace.md D1-D9, S): the bot sees Might on itself in both. The merc's share (V run in game.exe) is absent in d2d.
- (f) on from clvl 2 in the profile ladder (aura list [Prayer(rest), HolyFire, Might, Thorns]); B2.

### 3.4 Holy Fire (102, reqlvl 6, aura) rank 4
- (c) fire damage per pulse (V run, roll min .. max - 1 in 256ths, same roll for all targets, minus resist): level 1 256..767 (1.0 to 3.0 points), 2: 384..895, 3: 512..1023, 4: 640..1151 (2.5 to 4.5), 6: 896..1407 (V run pm_dmg.py). Synergy Resist Fire x par8 + Salvation x par7 (D). Radius 6 + level - 1 subtiles (1.2 to 2.2 cells). Not in town, not through walls (line of sight) (V run, palaura-trace.md).
- (d) per monster: 1.0 dps (L1), 1.75 (L4), 2.25 (L6) (I: 2.0 / 3.5 / 4.5 points average per 2 s). Compared with Might 4 (+4.5 dps on an 8.6 dps base at clvl 6) it needs about 3 monsters inside 1.8 cells to match. Maxroll and kolbot buy it at clvl 6 (V as sources, mining-2 section 3.2); the model does not support that for single-target kill speed. U for the live pack sizes; the AoE term wins in a dense pack.
- (e) d2d: every living monster within hypot x 5 <= radius, no filter / line of sight / town rule, the missile_blow roll (S); element damage numbers equal on 8,640 cases (V run).
- (f) ladder position B2.

### 3.5 Smite (97, reqlvl 1, S1; srvdofunc 150) rank 5
- (a) S1: 12 ticks (PAS11HS). (b) 2 mana, paid in the do (V read, no start function). (c) shield min / max (Buckler 1-3) + Holy Shield stat 17 / 18, x (100 + 15 + 15 (lvl - 1) + Str x 1) / 100; hit bit forced, no AR, no block (V read); stun min(250, 15 + 5 (lvl - 1)) frames, knockback ResultFlags 8. Needs a shield (returns 0 without).
- (d) clvl 12: 11.3 dps (I) against 16.5 Attack; the stun is the value (0.6 s at level 1, 4.4 s at level 20).
- (e) D8: d2d lets block cancel Smite (combat.hpp:499, S, buildable: player_blow with swing.smite against block > 0). (f) Out of the ladder; one v2 point at clvl 10. B4.

### 3.6 Holy Bolt (101, reqlvl 6, SC) rank 6 (burst, undead only)
- (a) SC, 16 frames (cast rate 16 ticks at FCR 0, I); missile holybolt Vel 20 Range 50. (b) mana 2.0 + 1/16 per level, paid at the action frame (D, V read). (c) magic damage 8-16 at level 1, 16-24 at 2, 24-32 at 3, 32-40 at 4 (V run, pm_dmg.py: 2048..4096, 4096..6144 ... in 256ths); only undead are damaged, others passed through; allies healed (V read; d2d lacks the heal).
- (d) 12 / 20 / 28 / 36 points a cast: 18.8 dps at level 1 for 3.1 mana/s, i.e. about 8 casts (96 points) from a clvl-6 pool of 22 mana (I). Good for a short undead burst, no sustained use.
- (f) no band; a prerequisite point only (clvl 20 in v2). B5.

### 3.7 Charge (107, reqlvl 12, SQ; srvstfunc 31, srvdofunc 67) rank 7 (movement)
- 9 mana at the start; ED 100 + 25 (lvl - 1) percent plus (Vigor + Might) synergy; AR +50 + 15 (lvl - 1) percent (D). A sequence skill: game.exe loses 30 attack-rate points in sequence mode (V run game side, skiasgame2 D1; not d2d). Used to close distance; not tabulated as dps. In no band.

### 3.8 Prayer (99, reqlvl 1, aura; rest) rank 8
- Heals every unit passing the friendly filter by 2 life at level 1 (512 in 256ths), +1 per level (6 at level 5, V run palaura_apply.py) once per pulse (2 s): **1 life per second at level 1**; costs 1.0 mana (+3/16 per level) on a pulse in which a life rose; state 85 stops mana regeneration on that pulse; with less mana than the cost it heals nothing (V run, pal-aura-trace.md). Range 16 + 2 per level subtiles reaches the merc. d2d: the player's heal only, free, no state 85 (S).
- (f) rest aura below 70 percent life (profile): right. Prayer and Might exclude each other, so Prayer stays a rest-only aura.

### 3.9 Later rows (clvl 16 and up; first points only)
- Vigor (115, reqlvl 18): velocitypercent dm56, stamina and recovery; the bot walks 68 percent of each kill, so Vigor is the strongest speed lever after clvl 18 (rule: walk-speed-trace.md, S). Holy Shield (117, reqlvl 24, 35 mana, SC): block and Smite bonus; Concentration (113): damage aura, Blessed Hammer's ED share (V run pal-skill-trace.md); Blessed Hammer (112, reqlvl 18); Defiance, Thorns, Resist Fire / Cold / Lightning, Blessed Aim: passives and defence, not kill speed. All D rows at `mpq-table.sh skills`.

## 4. Damage per second, one monster (I; `python3 tools/emu/pm_dps.py table`)

Short Sword 2-7, Dex 21, Str per the v2 stat rows (34 at clvl 6, 46 at 12, 50 at 15), monster level = clvl, defence 15 (and 60 in brackets), v2 skill points (Might 4 / 8 / 8, Sacrifice 1, Zeal 1 / 4):

| clvl | Attack (Might on) | Sacrifice 1 (life price per s) | Zeal game (d2d) | Smite |
|---|---:|---:|---:|---:|
| 6 | 13.1 (9.2) | 25.3 (18.5) at 2.0/s | n/a | n/a |
| 12 (Zeal 1) | 16.5 (11.5) | 28.7 (21.0) at 2.3/s | 23.8 (17.5) | 11.3 |
| 15 (Zeal 4) | 16.7 | 29.0 at 2.3/s | 31.8 (18.2) | 11.3 |

With a Mace 3-10 at clvl 12, Zeal 4: Attack 23.8, Sacrifice 41.5 (3.3/s), Zeal 45.3 (d2d 25.9). A pack changes the Zeal and Holy Fire rows upward (Zeal retargets inside the cast, Holy Fire hits everyone in range).

## 5. The order of skill points and the attack choice, clvl 1 to 20 (I; `pm_dps.py order`)

Damage only (the model picks Sacrifice whenever it can, with Might 1): clvl 1 Attack 8.0 dps; clvl 2 Sacrifice 1 = 20.2 (x1.9 over Might-only); clvl 6 {Might 1, Sacrifice 5} 27.6 (x1.95); clvl 12 {Might 1, Sacrifice 11} 34.9 (x1.90); clvl 20 44.5 (x1.85). So **damage per point: Sacrifice +15 percent ED per point against Might's +10; both on the same additive sum.**

With a cap on the life price the sensible ladder (U for the potion economics, I for the rest):
1. clvl 2-3: Sacrifice 1 and Might 1 (both ~doubling and +32 percent). Attack choice from clvl 2: Sacrifice, with the price (8 percent of the HP of what it kills; a 30-life Fallen costs 2.4 life, while a plain Attack kill lets it hit you about 2.3 s at ~1.3 dps = 3 life: break-even on weak monsters, a gain on strong ones; I).
2. clvl 4 to 11: Might until 4 to 5 points (marginal +0.65 dps a point at 13 dps), then Sacrifice 2 to 5 while potions hold (each point +1.2 dps and +0.1 life/s).
3. clvl 12 to 15: Zeal to 4 (23.8 -> 31.8, no life price); attack choice: Zeal once it beats the Sacrifice you have (Zeal 4 > Sacrifice 1 only), Sacrifice if you kept investing.
4. clvl 16 to 19: the v2 table spends them on Holy Fire; the model prefers Might or Zeal ED synergy (Sacrifice x12 each) over Holy Fire at 1.75 dps per monster (I).

v2 against this: it matches steps 2-3 apart from Sacrifice at 1 point with no casting; the missing part is B1 (cast it) and the Holy Fire points (B2).

## 6. Deviations d2d against game.exe (this card)

| id | rule | tag | differential |
|---|---|---|---|
| D1 | Zeal strike spacing: game.exe action frame ticks (7 at rate 256), d2d full swing (15) | game V run (fields I), d2d V run | built (zd_*), 24 of 24 gaps differ; no fix built; no PR candidate (pal-zeal-trace.md) |
| D2 | an idle tick between held swings in d2d (Attack 16, Zeal casts 16); game U | d2d V run, game U | buildable, drill H1 |
| D3 | Smite blockable in d2d, never in game.exe | V read both (D8 of the earlier trace) | buildable: `player_blow` with `swing.smite` against block > 0 |
| D4 | friendly auras: d2d reaches the caster only (no merc, party, pets), no mana for Prayer, no state 85 | S, V run game side (palaura-trace.md D1-D9) | buildable (apply tests) |
| D5 | Holy Bolt heals allies in game.exe; d2d only filters undead | V read, S | not built |
| D6 | Holy Fire/Shock enemy hit in d2d has no line of sight / town / filter rule | V run game, S d2d | buildable |

## 7. Bot findings (described, not changed)

- **B1** The attack ladder has no Sacrifice: `internal/combat/classdata/profiles.json` Paladin bands ([Blessed Hammer, Zeal, Attack] from 18, [Zeal, Attack] from 12, [Attack] below) and `internal/plandata/build-paladin-normal30-v2.json` (one Sacrifice point at clvl 3 "never cast before Zeal"). The model says x1.9 damage from clvl 2 for 8 percent of the damage as life; whether the net life cost is acceptable is U (drill C1, plus a soak with potion counts).
- **B2** `profiles.json` Paladin `aura` ladder is [Prayer (rest), HolyFire, Might, Thorns]; `internal/combat/aura.go:96-117` (`wantAura`) returns the first learned, reachable, non-immune entry, so a Paladin that has learned Holy Fire (v2 does at clvl 16 to 19, the mining proposal at clvl 6) never fights under Might. By section 3.3 / 3.4, Might should come first below about 3 enemies in reach. Reordering is data (the ladder is JSON) but this card did not change it.
- **B3** Zeal measurements are d2d-paced (pal-zeal-trace.md B1, B2): conservative in kill time by 1.74x for a 5-hit cast and mana per second 1.74x too low.
- **B4** Smite leaves the ladder correctly (11 dps), but the opener use on bosses (always hits, stun 0.6 s at level 1, no block) is not modelled; palskill-trace.md B1 (no shield check) stands.
- **B5** Holy Bolt is a prerequisite only (v2 clvl 20); the profile never casts it, correct for sustained fights, a missed burst against Act 1 undead (Normal Zombie / Skeleton packs): 18.8 dps at level 1 for 3.1 mana/s (I).
- **B6** Held-button attack: d2d repeats a plain Attack every 16 ticks; a kill-time model should use 15 for the game (D2, U).

## 8. PR candidates

None. The one proven d2d difference of this card (D1) rests on a new harness and a fix that is not built; D3 and D4 are earlier-trace items with no after-run. Nothing was added to docs/upstream/pr-queue.md and nothing was posted.

## 9. Drill recipe (fork verbs of docs/forkdbg.md; nothing run)

Z1 to Z4, H1: pal-zeal-trace.md. C1 (Sacrifice price): `debug points 96 1`, a dummy with life 50 and a 500-damage swing through the Sacrifice cast: expect life loss 4, not 40 (min of the damage and the target's life; U); C2: player life 10, hit a healthy foe: dies. M1 (Might + merc): Might 1 on a Paladin with a hired merc in 16 subtiles, read the merc's damage percent via `dump stats`: game.exe gives it, d2d does not (D4). S1 (Smite on a blocker): `debug mon set <id> block 75`, cast 97: d2d blocks, game.exe never (D3). HF1 (Holy Fire period): stand one cell from a 1000-life dummy under Holy Fire 4 for 20 s with `step 25`: 10 pulses of 2.5 to 4.5 points (if the live numbers show a shorter period, the D row of this card is wrong). P1 (Prayer): hurt Paladin and merc, Prayer 1, `step 50`: both +2 life, 1.0 mana paid (game.exe); d2d: the Paladin only, no mana.

## 10. Matrix-cell updates (proposed, for a future Paladin matrix doc or the Sorceress matrix section 5)

- Row "Sacrifice": "ED 180 + 15/lvl, AR +20 + 7/lvl, 0 mana, life price 8 percent of the damage dealt (V read); x1.9 dps over Attack at clvl 2 (I); bot never casts (B1)".
- Row "Zeal": "game 7 ticks per strike vs d2d 15 (V run both); 5-strike cast 43 vs 75; 2 mana per cast (V run); first points: ED 0 below level 5".
- Row "Holy Fire": "1.0 dps per monster at level 1, 1.75 at 4 (V run damage, 50-frame pulse D + V read); not better than Might below about 4 enemies (I); ladder order B2".
- Row "Might": "+40 + 10/lvl percent damage, caster / party / merc / pets (V run)".
- Row "Smite": "always hits (V read), shield damage 1-3 (Buckler), about 11 dps at clvl 12 (I), stun 15 + 5/lvl frames".
- Row "Prayer": "1 life/s at level 1, +1 per level, 1.0 mana per pulse on a hurt pulse (V run); rest-only is right".

## 11. Limits and dead ends

The dps figures are an arithmetic model (I): the to-hit, damage and swing rules are V / S, but Might's additivity, the Dex and Str the bot actually has, the monster defence and the life price's cap rule are assumptions listed above. The Holy Fire period (50 frames) and the Zeal rollback are read from the tables and the game function but not seen in a live game. No monster HP table was used (the monstats columns of the dump were not aligned reliably); kill times follow from HP / dps. Charge, Vengeance, Blessed Hammer, Holy Shield have no dps row here. d2d side of the Zeal run used the scratch copy of d2d (no commit id); the upstream read tree at e36922ea has the same rule at fight.cpp:2293.

## 12. Re-run

`python3 tools/emu/pm_dps.py table` and `order` (no host needed); the V run pieces: `D2_LABRUN_HOST=<host> tools/emu/run-on-worker.sh pm_dmg.py`, `zd_build.sh`, `zd_d2d.sh`, `zd_game.py` (pal-zeal-trace.md).
