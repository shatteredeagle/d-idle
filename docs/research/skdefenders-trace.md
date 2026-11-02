# Hirelings and players as defenders against resistances: game.exe 1.14d vs d2d vs the bot (decomp board item 17, SKDEFENDERS)

> status: card skdefenders, 2026-11-02, offline (no game, no d2d slot, nothing pushed or posted). Extends docs/research/sk-resist-trace.md (monster defender, SKRESIST) and docs/research/mercstats-trace.md Q7 (hireling stats, first read of the defender branch). Question: for a hireling or a player that is hit, which function decides "plain monster", what do the penalty, the cap, the floor, the flat reductions, absorb and the poison fields do, does a curse or aura apply in full, does d2d match, and does the bot's reading of a mercenary's resistances need a change?

Tags: **V read** read in the disassembly (upstream's `disasm.py` on the patched 1.14d game.exe; addresses given); **V run** game.exe's own functions run natively under the emulator on the worker host (counts and seeds given); **D** the patched 1.14d tables as game.exe's MPQ stack serves them; **S** d2d source at upstream main or an earlier trace of ours (named); **I** inference; **U** not verified. "Real d2d" means the compiled `rules::monster_blow` (components/rules/combat.hpp), called by `tools/emu/skdef_test.cpp`. No disassembly is pasted; functions are named `FUN_xxxxxxxx`.

## Answer in eight lines

1. **"Plain monster" is decided by `FUN_0063ee90`** (a unit of type 1 whose class is 0x10f, 0x152, 0x167, 0x230 or 0x231 returns 1, 2, 3, 4, 4; anything else, and any player, returns 0): V run over every class 0..0x3ff. `FUN_0057c1e0` stores "defender is type 1 and the test says 0" in the hit context (+0x14); a player is never plain, a hireling is never plain (V read 0x57c212..0x57c231).
2. **A hireling reads the Normal column.** `FUN_00573cb0` forces the difficulty index to 0 when the unit is one of the five hireling classes (V read 0x573df0..0x573e03); the hireling rows' resistance cells are blank (D; Act 5's two classes carry ResPo 70 in all three columns), and the level-up `FUN_00572840` then sets stats 0x27 / 0x29 / 0x2b / 0x2d to hireling.txt `Resist + Resist/Lvl x d / 4` (toward 0, at least 0; V read 0x572a77..0x572abd; V run in mercstats-trace.md, 2,940 cases).
3. **The hit-time rule for a player or a hireling is `FUN_0057be00`**: pierce at any value (also at 100+), then the DifficultyLevels ResistPenalty (expansion) or -20 / -50 (classic by game byte +0x6d) for every stat except 0x24 and 0x25, then the cap (75 + the max stat, at most 95; physical 50; magic uses its max stat 0x26), then the floor -100 for all six types. The stat holds the RAW sum; the penalty is added only here (V read; **V run: formula vs game.exe, 111,232 cases per seed, seeds 1 and 7, 0 mismatches**).
4. **Curses and auras apply in full** (no one-fifth rule) on a player or a hireling: `FUN_005c3540` returns the value unchanged for type 0 and for a hireling class (V read; V run, `adj` column, same cases, 0 mismatches).
5. **Order inside the hit** (`FUN_0057bf80`): the flat reduction (stat 0x22 physical, 0x23 for magic / fire / lightning / cold, in 256ths) comes off the damage FIRST, then the resistance, then absorb (V read; V run 111,232 cases per seed). d2d does it the other way round (D1).
6. **Absorb** (item stats 142..149, `FUN_0057bf10`) heals the absorbed part, percent capped at 40, then the flat part (V read; **V run, 100,000 cases per seed, 0 mismatches against the read rule**). d2d has no item absorb (D3, S).
7. **Poison**: the record's poison field (+0x28) is cut by stat 0x2d (penalty, cap, floor), its length (+0x2c) by stat 110 with penalty and cap 75; the state is stored as stat 0x4a = minus the per-tick amount (`FUN_0057ac50`, V read). The two cuts equal d2d's (V run, 0 differ in 26,628 / 26,748 core cases).
8. **d2d**: the resistance, penalty and cap of the four elements and the poison cuts are equal to game.exe; it deviates on the order of the flat reductions (D1), has no magic resistance for a defender (D2), no absorb (D3), takes no damage-resist curse on the player (D4), floors nothing on physical (D5, unreachable) and differs above 4,096 points per hit (D6, MulDiv). **Bot**: `PlayerResists.Effective` caps before the penalty (B1); the bot reads no mercenary resistance at all (B3), so nothing to change in a read, only in the formula. No PR candidate (section 7).

## 1. Questions answered (Q1..)

| id | question | answer | where | tag |
|---|---|---|---|---|
| Q1 | Which function decides "plain monster"? | `FUN_0063ee90(unit)`: unit type 1 and class 0x10f (Rogue, ret 1), 0x152 (Desert, 2), 0x167 (Iron Wolf, 3), 0x230 and 0x231 (Barbarian, 4); else 0. Called by `FUN_0057c1e0` for the attacker (ctx +0x10) and the defender (ctx +0x14): +0x14 = type 1 AND test 0. Also called by `FUN_00573cb0` (spawn), `FUN_005c3540` (curse), `FUN_0057c060` (damage scale). | 0x63ee90..0x63eee5; 0x57c212..0x57c231; call sites from `skdef_callers.py` | V read + V run (`skdef_tbl.py`: all 1,024 classes, five nonzero, a type 0 unit 0) |
| Q2 | Where does the Normal column read happen? | `FUN_00573cb0`: difficulty = game byte +0x6d clamped to 0..2, then 0 when `FUN_0063ee90(unit)`; level (0xaa), ToBlock (0xa3) and the six resist cells (0x144.. stride 0x6) are read at that index. | 0x573dc1..0x573e03, 0x573e83..0x573eed | V read |
| Q3 | What are a hireling's base resistances? | The five hireling MonStats rows have blank ResDm, ResMa, ResFi, ResLi, ResCo; Act 5's two rows (act5hire1 / act5hire2) have ResPo 70 in Normal, Nightmare and Hell. `FUN_00572840` (called from four sites, 0x533eb2, 0x56abc2, 0x5733f3, 0x57e950: creation and level-up, I) SETS stats 0x27 / 0x29 / 0x2b / 0x2d (all four the same value) to `max(0, Resist + Resist/Lvl x d / 4)`, d = level - the row's Level; so the Act 5 ResPo 70 is overwritten once the level-up has run (U whether it always runs before the first hit). Stats 0x24 / 0x25 stay at the blank cell (0). Value by class, difficulty and level: section 2 (D). | 0x572a77..0x572abd; MonStats and Hireling tables | V read + D; value V run in mercstats-trace.md |
| Q4 | What does a hit do to a player or hireling resistance? | `FUN_0057be00` (row of the descriptor table 0x732980, ctx edi): total = merged stat (`FUN_00625480`); pierce stat (333 fire, 334 lightning, 335 cold, 336 poison and poison length) subtracted when total < 100 or the defender is NOT plain; for a NOT plain defender and any stat except 0x24, 0x25: add the penalty (game +0x70 expansion: DifficultyLevels row +0 = 0 / -40 / -100 (D); classic: byte +0x6d 1 -> -20, 2 -> -50); total > 0 and not plain: cap = 75 + max-stat total (rows with a max stat: magic 0x26, fire 0x28, lightning 0x2a, cold 0x2c, poison 0x2e; poison length has none) at most 95, physical (0x24 row, no max stat) 50; total <= -100: -100 (every defender, every stat); physical with the attacker under state 0x2f and an undead defender: 0. | 0x57be00..0x57bf05 | V read + **V run** (formula vs game.exe: 111,232 cases per seed x2, 0 mismatches; `skdef_resist.py rule`) |
| Q5 | Does a curse or aura on a player or hireling take a fifth? | No: `FUN_005c3540` returns the value unchanged when the unit is type 0 or a hireling class, whatever the base; the fifth applies only to a plain monster with base >= 100 (sk-resist-trace.md Q6). | 0x5c3540..0x5c35a3 | V read + **V run** (`adj`, same cases: 0 mismatches) |
| Q6 | Which skills put such a curse or aura on a defender? | By Skills.txt: Lower Resist (91, player curse), Conviction (123, player aura, srvdofunc 66) lower fire / cold / lightning resistance; Resist Fire / Cold / Lightning (100, 105, 110) and Salvation (125) raise it; Amplify Damage (66) and Decrepify (87) change damageresist. **No monster skill carries an elemental resist aurastat**; monsters cast Amplify Damage (Succubus Witch rows, 9) and Decrepify (Doom Knight 3, Baal's throne, the dkmag rows, 5), Weaken (10) at players and hirelings. | Skills.txt `aurastat1..6`, MonStats `Skill1..8` | D |
| Q7 | Flat reductions and the order | `FUN_0057bf80` (mode 0, i.e. no resist-ignoring flag): field - flat (stat 0x22 for the physical row, 0x23 for magic / fire / lightning / cold; the descriptor's +0x18 index 1 / 2; each << 8, and for 0x22 scaled by record +0x54 / 1024 when that is positive: not run, U), field <= 0 -> 0; then res >= 100 -> 0, res != 0 -> MulDiv(field, 100 - res, 100); then absorb. | 0x57bf80..0x57c054, 0x57c27d..0x57c2a4 | V read + **V run** (`dealt`: 0 mismatches) |
| Q8 | Absorb | `FUN_0057bf10` after the resistance: percent stat (146 magic, 142 fire, 144 lightning, 148 cold) capped at 40, taken only when > 0: absorbed = MulDiv(dmg, pct, 100) goes to record +0x48 (healed) and off the damage; then the flat stat (147, 143, 145, 149) << 8, at most what is left, likewise. The heal is applied to the defender's life before the damage (0x57c844 / 0x57c852). | 0x57bf10..0x57bf75, 0x57c844..0x57c874 | V read + **V run** (100,000 cases per seed, 0 mismatches vs the read rule: net life change, healed, damage left) |
| Q9 | The poison fields | Row 8: field +0x28 (poison damage, stat 0x2d, max 0x2e, pierce 336) and row 7: +0x2c (length, stat 110, no max stat, pierce 336) each go through `FUN_0057bf80` like the elements (penalty, cap 75 + max or 75, floor). Then `FUN_0057ac50(attacker, damage, length)` stores the state (state 2) with the length added to the unit's frame counter and stat 0x4a = -damage; per-tick effect of stat 0x4a not traced (U). The hit's immediate total (+0x4c) ALSO adds the poison field (0x57c3a5..0x57c3b7), so the oracle's life drops by one per-tick amount at the hit itself (V run: dealt = pdmg in every poison case). Whether a live poison hit really takes that amount at once: U (D5). | 0x57c28a..0x57c3d0, 0x57ac50..0x57ad35 | V read + V run (the fields); I (the live effect) |
| Q10 | Damage scale by who hits whom | `FUN_0057c060` (percent applied to the five damage rows when it is not 100): attacker or defender missing, or the same unit: 100; player hits player: 17; a hireling or flagged summon hits a player: 17; hireling hits hireling: 25; hireling hits a unit of the monster-class flag: DifficultyLevels row +0x38; an attacker with the `FUN_0063edc0` class flag against a defender with unit-flag 0x80000000: 400 (200 when the defender is a hireling). A plain monster hitting a hireling or a player: 100. The 400 / 200 case is not understood (U). | 0x57c060..0x57c13e | V read (100 for the monster-on-defender case); U for the flagged cases |
| Q11 | Does d2d treat the merc / player as a not-plain defender? | Yes, by construction: the player and the merc are separate paths (Character::panel, Fight::merc_fighter) that apply `clamp(sum + penalty, -100, min(75 + max, 95))` to fire, lightning, cold, poison and poison length (stat 110, -100..75). Same arithmetic as Q4 for those five (V run through the line copy, section 4). | components/game/character.hpp:250-258, fight.cpp:1927-1932, :555-558 | S (code) + V run (copy) |

## 2. The rules as a formula and the numbers they give

```
raw[s]   = the unit's merged stat total (items + level-up base; curses and auras added in full)
r        = raw - pierce                       (always for a player / hireling; a plain monster only when raw < 100)
r       += ResistPenalty                      (stats other than 0x24, 0x25; 0 / -40 / -100 expansion, 0 / -20 / -50 classic)
r        = min(r, cap) if r > 0               (cap = min(75 + max stat, 95); physical 50; poison length 75)
r        = -100 if r <= -100
dealt    = (dmg - flat) x (100 - r) / 100  (r >= 100: 0)   then absorb: pct (<= 40) of that, then flat
```

A hireling's own resistance (D: Hireling.txt, `tools/emu/skdef_merc.py`; value = `max(0, Resist + Resist/Lvl x d / 4)`, then the rule above with the difficulty penalty, cap 75; selection of the row by the d2d rule, merc_stats, S):

| class (difficulty) | row Level | own resist at row level, +10, +20, +30 levels -> after penalty and cap |
|---|---|---|
| Rogue (Normal) | 3 | 0, 20, 40, 60 (unchanged, under the cap) |
| Rogue (Normal) | 36 | 66, 83, 101, 118 -> 66, 75, 75, 75 |
| Rogue (Nightmare) | 36 | 63, 80, 98, 115 -> 23, 40, 58, 75 |
| Rogue (Hell) | 67 | 115, 127, 140, 152 -> 15, 27, 40, 52 |
| Desert (Hell) | 75 | 136, 146, 156, 166 -> 36, 46, 56, 66 |
| Eastern Sorcerer (Hell) | 79 | 132, 142, 152, 162 -> 32, 42, 52, 62 |
| Barbarian (Hell) | 80 | 142, 152, 162, 172 -> 42, 52, 62, 72 |

(Normal rows from level ~40 on are at the 75 cap; Nightmare rows start at 23..43 after the penalty and reach the cap about 30 levels later; Hell rows stay under 75 almost always, so gear resistance counts there. Items add to the raw total before the penalty.)

## 3. d2d (S, upstream main; same arithmetic in the sorcrest checkout)

- Player: `Character::panel` (components/game/character.hpp:250-258): `res = clamp(sum + penalty, -100, min(75 + max stat, 95))` for fire, lightning, cold, poison; no magic resistance; damage reduced % `min(stat 36, 50)`; flat as stats 34 / 35 (combat.hpp:152). Poison length reduction `clamp(stat 110 + penalty, -100, 75)` (fight.cpp:555-558).
- Merc: `Fight::merc_fighter` (fight.cpp:1927-1932): `clamp(merc_st.resist + gear + penalty, -100, min(75 + max, 95))`, `merc_stats` (rules.hpp:1672) the level-up value.
- Monster blow on a defender: `rules::monster_blow` (combat.hpp:403-436): physical `phys x (100 - dr_pct) / 100 - dr_flat`, fire / lightning / cold `resisted(roll, res) - mdr`, magic `roll - mdr` (no resistance), poison `resisted(roll x 10, res[3])` for `resisted(dur x 2, plr)` ticks; whole points. The merc and thorns paths use the same order (fight.cpp:1749, :1774, :1857, :1869).
- Curses on defenders: the merc's damage-resist curse only (`merc_foe`, fight.cpp:4451-4457, added to dr_pct after its cap of 50); the player's curse is kept for the AI states and the HUD (:587, fight.cpp:4475) and changes no resistance (S: every use of `curses[0]` read).
- Item absorb stats: none (the only "absorb" in the rules is Bone Armor / Cyclone Armor's pool, fight.cpp:1344-1356).

## 4. Deviations (d2d vs game.exe)

Counts are from section 5 (seeds 1 and 7, in that order; 26,628 / 26,748 "core" cases, i.e. player and hireling, no curse, no pierce).

| id | rule | game.exe | d2d | tag | reach in 1.14d | differential |
|---|---|---|---|---|---|---|
| D1 | flat reduction vs resistance | (dmg - flat) x (100 - r) / 100, flat in 256ths first (`FUN_0057bf80`) | dmg x (100 - r) / 100 - flat (monster_blow:415, :421-425; merc fight.cpp:1749, 1774) | **V run, real build**: 2,659 / 2,721 core cases differ in `dealt` with a flat reduction set and nothing else different (seeds 1 / 7; section 5) | Players and mercs with Damage Reduced / Magic Damage Reduced gear (stats 34 / 35) taking elemental or physical hits at a resistance above 0: d2d takes LESS than game.exe by flat x r / 100 per hit (r 75, flat 20: 15 points) | exists: `skdef_resist.py game` vs `test_def --defend` (this card) |
| D2 | magic damage on a defender | stat 0x25 with penalty none, cap 75 + stat 0x26, floor, then flat 0x23 first | `roll - mdr`: no magic resistance at all | **V run, real build**: 5,279 / 5,305 core cases differ in `dealt` (every magic case with a nonzero resistance, hireling 2,624 / 2,660, player 2,655 / 2,645) | players or mercs with a magic resistance stat (0x25) on gear: d2d deals the full roll (I: which monster attacks use the magic type was not listed) | exists (same harness) |
| D3 | item absorb (stats 142..149) | after the resistance: pct (<= 40) then flat, healed to the defender | none | game side **V run** (100,000 cases per seed vs the read rule); d2d absence S (grep of components/) | players with absorb gear (flat and percent) | could be built once d2d has the rule: oracle `skdef_resist.py agame`, the d2d side is `monster_blow` after a port |
| D4 | damage-resist curse on the player (Amplify Damage -100, Decrepify -50) | in full on the total (stat 0x24 total = raw + curse), floor -100, so a cursed player takes up to x2 | merc: yes (`merc_foe`); player: none | game side **V run** (`adj` = value); d2d S (read) | the Succubus Witch rows (Amplify Damage) and the Doom Knight 3 / dkmag / Baal's throne rows (Decrepify) casting at the player (D, MonStats skill cells; which difficulties spawn them not checked) | could be built: `test_game` scenario with a cursed player hit by a monster (Fight::world); oracle: `FUN_005c3540` + `FUN_0057c6c0` as `skdef_resist.py game` |
| D5 | physical floor and cap order | total (items + curse) capped at 50 when > 0, floored at -100 | `min(items, 50)`, the merc curse added after, no floor | **V run, real build**: physical effective resistance below -100 (1,270 / 1,247 core cases, all of them `floor`) | unreachable as far as seen (I: no negative-DR source found; Amplify gives -100 on a raw 0, which both sides handle) | yes (same harness) |
| D6 | hit above 4,096 points (MulDiv) | `FUN_00483360` divides first when the value in 256ths is above 0x100000: (fine / 100) x (100 - r), so the floor in points can be 1 below the exact one | exact whole points (floor of the exact product) | **V run, real build**: 8,292 / 8,275 core cases (every one with roll - flat above 4,096 points; in the cases listed game is exactly 1 point below d2d) | hits above 4,096 points: Hell boss spells only (I); the same MulDiv gap was found for monsters in skill-damage-trace.md | yes (same harness) |
| D7 | poison hit's immediate amount | the hit total (+0x4c) includes the per-tick poison field (V read + V run: dealt = poison field after resistance) | `monster_blow.damage` excludes poison; Fight applies it per tick (U whether a first tick lands at once) | oracle side V run; live effect and d2d tick path U | every poison hit on a player or merc (a per-tick amount, a few points) | no: needs a live read (drill R4) and d2d's Fight poison path |
| D8 | player damage scale `FUN_0057c060` | 100 for monster -> player / hireling | not modelled (none needed) | V read | 17 / 25 / 400 / 200 cases are PvP, merc-vs-merc or the unknown flagged pair | n/a (U for 400 / 200) |

Matches (V run through the line copy and the real `monster_blow`, 0 differences in the core): effective resistance of fire, lightning, cold and poison for player and hireling (penalty by difficulty and expansion / classic, cap 75 + max stat up to 95, floor -100), the poison damage cut and the poison length cut. Port cross-check: the python copy of d2d's lines (`skdef_resist.py port`) equals the real build in all 98,675 / 98,727 cases (`d1.tsv` and `p1.tsv` have the same md5), so the d2d column is d2d.

## 5. The differential (V run)

Oracle `tools/emu/skdef_resist.py` runs next to upstream's `tools/emu` (`emu.py`, `life_apply.py`, here the skmiss checkout on the worker host). Per case it runs natively `FUN_005c3540` (the curse value on the defender), `FUN_0057be00` (effective resistance, context +0x14 set as `FUN_0057c1e0` sets it for the defender's kind) and `FUN_0057c6c0` (the whole hit: flat, resistance, MulDiv, absorb, life) with the defender a player (unit type 0), a hireling (type 1, class 0x10f; `FUN_0063ee90` hooked by class, run natively over every class by `skdef_tbl.py`) or a plain monster (control, game side only), the attacker a plain monster, difficulty byte and expansion flag set from the case, DifficultyLevels ResistPenalty from the table (0 / -40 / -100). Hooked (not the claim): the stat getters (a dict), `life_apply.py`'s side-effect hooks (events, states, logging), the unit-kind predicates other than `FUN_0063ee90`. Case = `id kind diff exp elem S M v pierce aux roll len` (field meanings in the script's header).

Cases (`gen N SEED`): a boundary grid of 11,232 (2 kinds x 3 difficulties x 2 expansion flags x 6 elements x 13 totals (-120 .. 120, around 0, 74..76, 95..100) x 3 max-stat values x 4 curse values) plus N = 100,000 random (totals -200..200, max stat 0..30, curse -200..+60, pierce 0..60, flat 0..60, roll 1..20,000 points, poison length 1..2,000; kind 2 = plain monster about one case in eight). xorshift32 on a 32-bit masked seed. 111,232 cases per seed, seeds 1 and 7.

| check | cases | result |
|---|---|---|
| game.exe vs the READ rule (formula `ref_line`; kinds player, hireling and plain monster): `adj`, `effres`, `dealt`, poison `pdmg`, `plen` | 111,232 per seed, seeds 1 and 7 | **0 mismatches** in all five columns, both seeds (`skdef_resist.py rule`) |
| absorb: game.exe vs the read rule (healed, damage left, net life change) | 100,000 per seed, seeds 1 and 7 | **0 mismatches** (`agen` / `agame` / `abscmp`) |
| game.exe vs the REAL d2d build (kinds player, hireling) | 98,675 (seed 1), 98,727 (seed 7) | all cases: effres 40,944 / 40,730, dealt 73,314 / 73,161, pdmg 7,506 / 7,546, plen 3,618 / 3,742 differ; the differences are the curse column (d2d keeps no curse on a player or hireling's resistances: D4, Q6 says no monster casts one) and pierce (a monster has no pierce stat in 1.14d: unreachable) |
| the same on the "core" (v = 0 and pierce = 0): effective resistance | 26,628 / 26,748 | 0 differ for fire, lightning, cold, poison and poison length; 1,270 / 1,247 differ, all physical below -100 (D5) |
| core: poison damage, poison length | 26,628 / 26,748 | 0 / 0 differ |
| core: damage dealt (floor of game's 256ths vs d2d whole points) | 26,628 / 26,748 | 17,500 / 17,547 differ, by cause (each case counted once, first matching cause): flat order D1 2,659 / 2,721 (fire, lightning, cold, physical, both kinds), magic without resistance D2 5,279 / 5,305, physical floor D5 1,270 / 1,246, MulDiv above 4,096 D6 8,292 / 8,275; no case left unexplained (the sums are 17,500 and 17,547) |
| port (`skdef_resist.py port`) vs the real build | 98,675 / 98,727 | 0 differ in every column |

Ledger count (docs/coverage/ledger.tsv, the F04 row's note; the row key already belongs to sk-resist-trace.md): 222464 cases (111,232 x 2 seeds, the rule check on game.exe).

md5 prefixes, seed 1 / seed 7: cases 2ae92703 / f7bab504; game.exe output fc67b79e / c5ded4f1; real d2d output (equal to the port's) 9b50ce2a / 13f7ce04; absorb cases 46c03a6d / 67d790e1, absorb game output ae1345cf / 890d81bc.

Commands (the worker host, scripts from this card's `tools/emu`; upstream checkout with `emu.py`):

```sh
tools/emu/run-on-worker.sh skdef_tbl.py            # descriptor rows, FUN_0063ee90 over every class
tools/emu/run-on-worker.sh skdef_build.sh          # compile skdef_test.cpp against upstream main's headers
tools/emu/run-on-worker.sh --ttl 3000 skdef_all.sh 100000 1 7   # gen, game.exe, port, diff vs real d2d, rule check, absorb
tools/emu/run-on-worker.sh skdef_merc.py           # the hireling resistance table of section 2
tools/emu/run-on-worker.sh skdef_callers.py 0x572840 0x63ee90   # call sites
```

Honest limits: the d2d `effres` column is a line copy of `Character::panel` / `Fight::merc_fighter` (those need a Character / Fight to run): the damage, poison and length columns are the REAL `rules::monster_blow`. Only the hit's floor is compared (game 256ths vs d2d whole points); the fraction d2d drops is not measured. The attacker is a plain monster (no PvP, no sanctuary, no record flags 0x100 / 0x200 / 0x400, record +0x54 = 0, no `FUN_0057c060` scaling other than 100).

## 6. The bot (our repo; described, not changed)

| id | finding | file:line | tag | impact |
|---|---|---|---|---|
| B1 | `PlayerResists.Effective` caps the raw value at 75 BEFORE adding the penalty (`max(min(v, 75) + p, -100)`); game.exe adds the penalty to the raw sum first and caps after (`min(v + p, 75 + max)`). Raw 100 fire in Hell: bot -25, game 0; raw 90 in Nightmare: bot 35, game 50. The max-resist stats (95 cap) are not read. The doctor line repeats it. | internal/combat/rules.go:308-312; internal/d2/difficulty.go:40 | V run (the rule, 111,232 x 2 cases) + S (our code) | the bot rates itself (ResistDanger, packDanger, hellCtx PoisonRes) worse than game.exe for any resistance above 75 raw in Nightmare / Hell: conservative |
| B2 | the repo already has the right order in two places (`ResistGate`: `min(v + pen, maxResist)`; `ResistValue`: `capRaw = 75 - pen`), so the same repo disagrees with itself. The open question in the TODO / `ReadContainsPenalty = false` ("is the penalty already in the stat read?"): at the hit the game reads the raw merged stat and adds the penalty itself (`FUN_0057be00` only reads; V read); so for the stat list the default `false` is the right one, for the character screen's number it is U (other code). | internal/combat/hard.go:29-33; internal/leveler/resistgate.go:33, :63; internal/gear/score.go:32 | V read (hit path), U (display) | the "applied twice" warning in hard.go is a non-issue if the read is the merged stat list |
| B3 | The bot reads NO mercenary resistance: `merc.Item.Res` is the sum of the four +resist values of an item, `MercState` has no resist field. Nothing to change in a read; if a later card reads the merc unit's stat list (0x27 / 0x29 / 0x2b / 0x2d) the values are RAW (base from the level-up plus gear plus any aura / curse in full): apply penalty then cap 75 (+0x28 etc.) as in section 2, per element, not summed. | internal/merc/types.go:71; internal/merc/gear.go:10-12 | V run (rule) + I (no reader) | the policy question below |
| B4 | The gear score counts resistance only in Hell (2 per point of the four-sum). By section 2 a merc's own resistance after the penalty is 23..58 in Nightmare (still under the cap, gear counts) and in Normal it reaches 60 at level 33 and sits at the cap from about level 46 (gear is worth nothing past the cap; in Nightmare that comes about 30 levels later): "Hell only" is right for Normal at higher levels, too narrow for Nightmare, and a per-element cap is missing. | internal/merc/gear.go:10-12 | D (table) + V run (rule) + I (policy) | a Nightmare merc gets no resistance gear it could use; ranking only |
| B5 | `MercClasses` lists 271, 338, 359, 561 and says 560 "appears in no hireling row and is deliberately absent". Hireling.txt indeed uses 561 only (D), but game.exe's class test names BOTH 0x230 and 0x231 (V run): a unit of class 560 would be a hireling defender too. | internal/d2/merc.go:6 | V run + D | none unless a unit of class 560 exists (I: never hired) |
| B6 | `hellCtx` takes poison RESISTANCE as the poison-defence measure; the game cuts the poison damage by stat 0x2d and the poison LENGTH by stat 110 (poison length reduction, own penalty and cap 75): a character with poison resist and no length reduction takes the long poison. | internal/combat/hard.go:38 | V run (two stats) + I (policy) | potion policy only |

## 7. PR candidate

**No PR candidate.** D1 and D2 are proven on the real `rules::monster_blow` by the harness of this card (before: D1 2,659 / 2,721 and D2 5,279 / 5,305 core cases differ per seed), but: the harness is new here (the rule of this board is the EXISTING differential), a fix touches several sites (`monster_blow`, `merc` hit paths in fight.cpp, `blast_taken`, the thorns paths) so a faithful "after 0" needs a patched build and a det-state check that this card (no fork checkout write, no d2d slot) cannot produce, and the effect of D1 is flat x resistance per hit. A follow-up card should do it with the independent verifier: before 2,659 / 2,721 (D1) and 5,279 / 5,305 (D2) -> after 0 on the same cases, D5 and D6 excluded or fixed alongside.

## 8. Drill recipe (lab binary, fork verbs; for a later live card, not run here)

Verbs (docs/forkdbg.md, docs/forkcheats.md, docs/d2d-fork.md): `debug difficulty`, `debug stat`, `debug goto`, `debug spawn`, `debug mon set`, `debug states set`, `merc hire`, `merc hurt`, `hit`, `dump stats` (raw stats in 256ths), `monsters`, `step N`. All ledgered, throwaway `--no-save` characters, hidden slot. Missing verbs the drills want: `debug curse <player|merc> <skill> <level>` (to apply Amplify Damage / Lower Resist as a monster cast would), and a way to give the player a stat (the `debug stat` form exists for the raw stat list: use ids 0x22 / 0x23 / 0x27 / 0x28, 142..149).

- R1 (penalty and cap order, Q4, B1): `debug difficulty 2` (Hell), set the player's stat 0x27 (fire) to 100 and max 0x28 to 0; `hit` / a Hell fire monster's attack with a known roll; the life lost must match r = 0 (game.exe), d2d before a fix also r = 0 (the panel clamp is right); then the bot's `Effective` would say -25 (B1) — read the bot's number from its log, no game difference expected. Repeat with raw 120 in Nightmare (r = 75 + max).
- R2 (merc penalty, Q3, section 2): `merc hire` a Rogue at Normal, then `debug difficulty 2`; the merc's resist on `dump stats merc` / its fire damage taken at a known roll equals `max(0, Resist + Resist/Lvl x d / 4) - 100`, floored at -100; Normal and Nightmare the same with the band row of section 2.
- R3 (flat order, D1): give the player stat 0x23 = 20 (flat magic reduction) and fire resist 75 (stat 0x27 75); a fire hit with a known roll 500 must take (500 - 20) x 0.25 = 120 (game.exe), d2d today 105. Physical: stat 0x22 = 20, DR 25: game (roll - 20) x 0.75, d2d roll x 0.75 - 20.
- R4 (poison hit, D7, Q9): a poison hit on the player with a known per-tick amount; read the life at the hit tick and every tick after: game.exe takes one per-tick amount at the hit (if Q9's reading is real), then one per tick for the length; d2d only per tick. A poison length reduction stat (110) of 50 halves the length; with the Hell penalty the same stat reads 50 - 100 = -50 (lengthened by half).
- R5 (curses on the player, D4): a monster Succubus Witch (Amplify Damage) in Hell next to the player: the player's physical damage taken doubles while the curse lasts (game.exe), d2d: unchanged for the player, doubled for the merc.
- R6 (absorb, D3): player with stat 142 (fire absorb %) 20 and 143 flat 10: a fire hit heals 20 % then 10, life goes UP when the damage is under the absorb.

## 9. Matrix-cell updates (docs/research/sorc-skill-matrix.md; proposed text for the merger)

| Row | Cell | New value |
|---|---|---|
| resist / immunity rows (Fire Bolt, Lightning, Cold: (a) resist, immunity) | defender side | player and merc as defenders: penalty then cap 75 + max, floor -100, curses in full, flat reduction before resistance, absorb after (V run, 111,232 cases x 2, skdefenders-trace.md); d2d matches the penalty and the cap for the four elements and the poison cuts, differs on the flat order (D1), magic resistance (D2), absorb (D3), the player's Amplify (D4) |
| rank 8 SKRESIST (status) | status | defender side traced (skdefenders-trace.md): no PR built; D1, D2 provable on the real build, fix needs a patched build |
| NM / Hell gate (SKRESIST) | resist rows | the penalty is hit-time: raw above 75 still counts in Nightmare / Hell; the bot's `Effective` applies the cap first (B1) |
| Energy Shield, Frozen Armor (absorb rows) | none | no change: those are state absorbs; item absorb stats 142..149 are a separate path (D3) |

## 10. Dead ends and not traced (U)

- First reading of `FUN_0057c060`'s 400 / 200 case (an attacker class flag against a defender with unit flag 0x80000000): what the two flags are (MonStats flag byte masks at 0x6ce280 / 0x6ce284 and the unit's +0xc4 bit 31) was not resolved.
- The record flags 0x100 / 0x200 / 0x400 (`FUN_0057c1e0` mode word: when set, positive resistances count as 0 and the flat reductions are skipped; set by an attacker's class flags from `FUN_0063e990` / `FUN_0063e940`): not run, not mapped to a skill or monster.
- Record +0x54 (scales the flat physical reduction by +0x54 / 1024 when positive; a per-frame or fractional hit): not run.
- The per-tick effect of stat 0x4a (the poison's life per frame) and what d2d's Fight does with the first tick: not traced.
- Sanctuary's undead DR bypass on a defender: unchanged from sk-resist-trace.md D4.
- Whether `FUN_00572840` always runs before the first hit on a freshly hired merc (the Act 5 ResPo 70 cell would otherwise be a base): not traced.
- The character screen's resistance number (penalty applied for display): other code, U.
- Fractions: d2d keeps a player's and merc's life in whole points for monster hits; game.exe in 256ths. Only the floor was compared.

## Shared-file lines for the merger

In `openspec/changes/skdefenders/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG), and for docs/decomp-queue.md (finished list, item 17):

```
- 17 defenders (SKDEFENDERS, 2026-11-02): plain monster = FUN_0063ee90 (classes 0x10f 0x152 0x167 0x230 0x231 are hirelings, V run over every class); a player or hireling takes pierce at any value, the penalty (not 0x24 / 0x25), the cap 75 + max (physical 50), the floor -100, curses in full, flat reduction before the resistance, absorb after (FUN_0057be00 / 0057bf80 / 0057bf10, V run 111,232 cases x 2 seeds, 0 mismatches; absorb 100,000 x 2, 0); d2d equal on penalty, cap and poison cuts; deviations D1 flat order, D2 no magic resistance (both V run on the real monster_blow), D3 no item absorb, D4 no Amplify on the player, D5 physical floor, D6 MulDiv above 4,096, D7 poison first tick (U); bot B1 Effective caps before the penalty, B3 no merc resist read, B4 gear score Hell only, B5 class 560; no PR candidate; docs/research/skdefenders-trace.md
```

Glossary (proposed, not written): "defender branch": the half of the hit resistance function (FUN_0057be00) taken when the defender is a player or a hireling (penalty, cap, pierce at any value) as opposed to a plain monster.
