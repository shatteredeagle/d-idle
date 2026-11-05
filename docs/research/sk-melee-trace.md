# The player's plain Attack (melee blow): game.exe 1.14d vs d2d vs the bot (decomp row 20, SKMELEE)

> status: current (2026-11-01, agent SKMELEE). Offline: no game, no d2d slot, nothing pushed or posted. Tags: **V read** = read in game.exe's disassembly (`disasm.py`) or in d2d's code at file:line; **V run** = game.exe's own function run under unicorn by `tools/emu/sk_melee.py`; **D** = 1.14d patched tables (ItemStatCost, ItemTypes); **S** = an upstream doc or an earlier trace; **I** = inference; **U** = unverified. d2d = upstream main 72132b99 (read worktree on the worker host). No game code is pasted here: function tags and short formulas only.
> status: deviations re-proven on the real d2d build, D7 narrowed, D8 added: docs/research/sk-melee2-trace.md (SKMELEE2).

## Summary

1. **The brief's starting point is a different function.** FUN_00647bc0 is a *skill's* physical minimum (Skills.txt MinDam with its level steps, HitShift and the weapon SrcDam share; a kick's own rule), used by missiles and the char panel (S skill-damage-trace.md row 1; V read). The plain Attack does not call it. The blow is do 1, **FUN_0056f070** (S upstream skills.md; V read).
2. **To-hit: same formula as d2d** for every attack rating over 0: 0 differences from the reading (V run), the d2d port differs only where the attack rating is 0 or less (d2d clamps it to 1) or the to-hit % is negative (rounding). Game.exe also has five gear terms d2d does not model (ignore target defense, -% target defense, AR vs demons / undead / monster type: D3).
3. **Damage with a weapon: same whole points as d2d** on 100,000-case runs (reading A of the gear stats, see Q3), but **d2d floors the physical to whole points while game.exe keeps 256ths** (D1): a landed blow deals on average about half a point more in game.exe. For example a 1-5 staff at 0 % (mean 3.0 in game.exe, 2.5 on d2d: arithmetic on the two rules) loses about 17 % of the swing.
4. **Two rules d2d lacks entirely:** Blunt weapons (Staff, Scepter, Wand, Club, Mace, Hammer: ItemTypes `blun` by equivalence) get +50 points in the damage % against undead (1.5x when the rest of the % is 0), plus gear stats 121 / 122 / 180 (D2, V run on FUN_0057b660); and **barehanded** blows get the whole Strength as a damage % and take jewelry +min / +max as the range, not on top of 1-2 (D4, V run).
5. **Open (U):** whether the weapon's own enhanced damage also reaches the player's stats 17 / 18 (which FUN_0057b420 adds to the %), and so counts twice, and whether a shield's damage reaches stats 21 / 22. Both are statlist aggregation questions (Q3); drill R1 / R2 settle them by reading the player's stats.
6. **No PR candidate**: no d2d deviation is proven on a real d2d build with an existing differential harness (the comparison here is against a port of d2d's rule).

## Q1 The call chain of one plain Attack (V read unless marked)

| Step | game.exe | What it does |
|---|---|---|
| Do 1 | `FUN_0056f070` | weapon class 0x26 (FUN_0062b400) returns at once; a bow / crossbow (FUN_0064f460) fires its missile (FUN_00645f00, S skills.md "Shots and ammo"); else the target (FUN_00553540), a zeroed 0x70-byte damage record, then the steps below |
| To-hit | `FUN_0057ec10(attacker, target, bonus = stat 325, range 0)` (0056f151) | both units player / monster, FUN_00554200 (a foe), a monster target passing FUN_005a4ee0 (not traced); reach FUN_00622c40; a player target in mode 3 (RN) is hit without a roll; else `FUN_0057d9b0(.., bonus, missile 0)`; then the defender's block / dodge (FUN_0057dfb0, can-block 1); result flags in record +4 |
| Pre-events | `FUN_005d3ac0` (0056f163) | walks the state table, and for each state the attacker has (FUN_00639df0) whose stat list (FUN_006256b0) names a skill (stat 350) with a row byte +0x44 of 1, FUN_005d3680 on the record (I: aura / state hit events; not traced further, U) |
| Build | `FUN_0057b7d0(attacker, target, record, 0, SrcDam 0x80)` (0056f178) | below (Q2); record flag 2 set first, so FUN_0057dbf0 does not build it again |
| Post | `FUN_005d3ba0`, then `FUN_0057dbf0(.., record, 0x80)` (0056f195) | apply: resistances / absorbs FUN_0057c1e0, the kill test on the 256ths life (stat 6 & ~0xff), takes-damage FUN_0057ca80 |
| Charge-ups | `FUN_0063a7b0(.., state 0x26)` -> `FUN_0056c2d0`, `FUN_0057d690` -> `FUN_005d5220`, `FUN_0057d4f0` | an Assassin's finisher release and the swing's follow-ups (S skills.md "Charge-ups") |

### To-hit: FUN_0057d9b0, attacker a player (V read; V run 100,000 cases)

- Attack rating `FUN_00622560`: stat 19 + (dex - 7) x 5 + CharStats ToHitFactor (+0x3c). d2d `make_fighter`: `dex x 5 - 35 + gains.to_hit + stat(19) + 224 per level` (combat.hpp:116): same.
- `FUN_0057d8a0` (gear terms, D3): stat 115 sets the defense to 0 against a non-boss monster (FUN_005a0180, FUN_0063e9f0, FUN_0063ee90 decide "boss"); stat 116 cuts the defense by MulDiv(def, min(116, 100), 100), halved against a boss; stat 123 adds to the AR against a demon (FUN_0063e940), 124 against an undead (FUN_0063e990).
- Percent = mastery to-hit (FUN_00645830(.., weapon, 0, 0), melee with a weapon only) + stat 119 + the bonus (stat 325, Assassin progressive to-hit; 0 otherwise) + stat 179 entries matching the target's MonType (FUN_006261d0 list, FUN_0057a830). AR' = AR + MulDiv(AR, percent, 100).
- Defense = FUN_006223f0 + stat 33 (melee). Chance = AR' x 100 / (AR' + DEF), x 2 x alvl / (alvl + dlvl), 5..95; hit when the attacker's seed (unit + 0x20) % 100 is under it. d2d `hit_chance` (combat.hpp:28): same arithmetic; `player_blow` (combat.hpp:470): `max(ar_base x (100 + ar_pct + swing.ar_pct) / 100, 1)`.

## Q2 The physical damage (V read; V run 100,000 cases)

`FUN_0057b7d0` (the damage build) for an attacker that is a player (or a unit FUN_006259b0 says is 2) striking a monster, before the roll: stat 120 is added to the target's defense (FUN_00627260 on stat 31, floored at 0); against a demon (FUN_0063e940) stat 121 joins the record's ED (+0xc); `FUN_0057b660`: against an undead, **+50 with a Blunt weapon (FUN_00629bb0(weapon, 57))** plus stat 122, added to the ED when the sum is over 0; `FUN_0057b6a0`: stat 180 entries matching the target's MonType. Then (unless record flag 1) `FUN_0057b420(attacker, 1, weapon 0, 0, 0, ED = record +0xc, flat = record +8, SrcDam)`:

- The weapon: FUN_00535bc0 (the hand the current skill uses). With one: min / max = the player's stats 21 / 22 (23 / 24 when FUN_0063d340 says the second hand is used), no floor yet. Without: min = stat 21 if over 0 else 1, max = stat 22 if over 1 else 2.
- Both << 8, plus stat 111 << 8 on both; min under 1 becomes 0x100; max at or under min becomes min + 0x100.
- One percent p = the ED argument + stat 25 + (with a weapon) str x StrBonus / 100 + dex x DexBonus / 100 (FUN_00629860 / FUN_006298a0, C division) + mastery damage (FUN_00645830(.., 1)); **without a weapon p += strength**. p is floored at -90.
- min' = min + MulDiv(min, **stat 18** + p, 100), max' = max + MulDiv(max, **stat 17** + p, 100); result = min' + flat + rand(max' - min') on the attacker's seed (top exclusive; no roll when max' <= min'), negative to 0, x SrcDam / 128 (MulDiv) when SrcDam is not 0x80. Returned in 256ths into record +8.
- Back in FUN_0057b7d0 (record +8): when the third argument is 0, the mastery crit (FUN_00645830(.., 2), a weapon only), critical strike (stat 337), deadly strike (stat 141) are separate rand(100) rolls on the attacker's seed, any one doubles and sets flag 0x2000. d2d: same order (combat.hpp:521).
- The record keeps 256ths; FUN_0057dbf0 / FUN_0057c1e0 take the life in 256ths (S monster-attack-trace.md, pr-missile-damage-fine-proof.md: stat 6 takes the fine value).

d2d (V read): `make_fighter` (combat.hpp:110-118) builds the weapon range in whole points as `mindam x (100 + weapon 18) / 100 + flat 21 + 111` (same truncation as op 13's MulDiv on the item), `<< 8`, the same floors; `phys_pct = max(25 + 343 + str x StrBonus / 100 + dex x DexBonus / 100, -90)` (barehanded: no Str / Dex term); `player_blow` (combat.hpp:496-506) `max(pct + swing.ed, -90)`, roll top exclusive, crit, `x srcdam / 128`, then **`phys = max(damage >> 8, 1)`**, whole points. No undead, demon, monster-type or stat 120 term anywhere in `player_blow` / `make_fighter` (grep of components/: none).

## Q3 What the player's stats 21 / 22 / 17 / 18 hold (V read in part; U)

- ItemStatCost (D): 17 / 18 `item_max/mindamage_percent` op 13 on maxdamage / secondary / throw (and mindamage ...), damagerelated 1; 21 / 22 / 23 / 24 / 25 / 111 damagerelated 1.
- `FUN_00626200` (a stat's value with its op terms, V read): op 13 adds MulDiv(the owner's base stat (FUN_006253b0), the op stat, 100) **only when the list's owner is an item** (type 4). So the weapon's own list holds stat 21 = base + base x ED / 100 + its flats.
- `FUN_00624fe0` (a stat's base in a list, V read): an extended list adds its children's values (their full list when extended), skipping a child flagged 0x40000000 for a damagerelated stat. FUN_00627910 (merge an item's list into a unit's) sets / clears that flag by its third argument (FUN_00627860 / FUN_00627a40); the weapon-switch slots 11 / 12 go elsewhere (FUN_006269f0). Which items are merged with the flag set was not traced.
- **Reading A** (d2d's model, used by the oracle): the player sees the weapon's op-13 share in stats 21 / 22 and stats 17 / 18 are 0 at the player. **Reading B**: stats 17 / 18 also reach the player (nothing in FUN_00624fe0 stops a weapon's damagerelated stat), so FUN_0057b420 applies the weapon ED a second time, on base + ED + flats. The code read so far does not exclude B; the community rule (weapon ED once, on the base) favours A (I). **U** until drill R1.
- Shield carrier: a shield's base damage reaches the plain Attack only if the shield's list carries stats 21 / 22 into the player's without the 0x40000000 flag (same mechanism, U, drill R2). d2d's `sum` never holds a shield's base damage (d2s items keep only their magic stats); the shield is read only for Smite (`smite_lo`, combat.hpp:135) and block. Dual wield: the second hand's 23 / 24 by FUN_0063d340 (V read, not run).

## Differential (V run, the worker host, upstream read worktree on 72132b99)

`tools/emu/sk_melee.py` runs game.exe's FUN_0057d9b0, FUN_0057b420 and FUN_0057b660 natively (stat getters and the item / unit helpers hooked, the seed draws native) against the reading and against a port of d2d's rule (not the real d2d build).

| Part | Cases (seed 7 / seed 1) | vs the reading | vs d2d's rule (port) |
|---|---|---|---|
| to-hit FUN_0057d9b0 (player at monster, melee) | 240 boundary + 100,000 random each | 0 / 0 | 44 / 36 (seed 7: 43 with AR' <= 0, 1 with a negative %; seed 1: 36 with AR' <= 0) |
| damage FUN_0057b420 (plain Attack: ED 0, flat 0, SrcDam 0x80; 85 % weapon, 15 % barehanded) | 100,000 each | 0 / 0 | whole points: weapon 0 of 84,877 / 0 of 85,124, barehanded 13,048 of 15,123 / 12,729 of 14,876; 256ths: 99,683 / 99,683 differ; mean game minus d2d a landed blow: weapon +0.4978 / +0.4991 points, barehanded +3.055 / +3.055 |
| undead FUN_0057b660 | 60 (exhaustive grid: undead x {none, non-blunt, blunt} x stat 122 {-60, -1, 0, 1, 50} x record ED {0, 35}) | 0 / 0 | d2d adds nothing |

Reproduce (the worker host):

```sh
cp tools/emu/sk_melee.py <upstream>/tools/emu/ && cd <upstream>/tools/emu
D2_MPQ_DIR=<mpq dir> <venv>/bin/python sk_melee.py --exe <game.exe> 100000 7
D2_MPQ_DIR=<mpq dir> <venv>/bin/python sk_melee.py --exe <game.exe> 100000 1
```

Case generator (seed masked to 32 bits): AR -40..2500, defense -30..1500, to-hit % in {0, 10, 25, 50, 100, 200, -10, -25}, mastery {0, 28, 61}, stat 325 {0, 30, 100}, levels 1..98; weapons 1-40 / +1-60 base, ED {0, 30, 50, 100, 200, 300}, flats 0..10, stat 111 0..2, stat 25 {0, 20, 100, -25}, Str / Dex 10..199, StrBonus / DexBonus pairs {100/0, 110/0, 0/100, 75/75, 0/0}, skill ED {0, 50, 150}.

## Deviations d2d vs game.exe

| # | Rule in game.exe | Rule in d2d | Tag | Effect | Differential |
|---|---|---|---|---|---|
| D1 | the physical stays in 256ths into the record and the life (FUN_0057b420 -> record +8 -> FUN_0057dbf0) | `phys = max(damage >> 8, 1)` per blow (combat.hpp:506); `Blow::frac` is only filled by missiles | V run (game side), V read (d2d) | about +0.5 point a landed blow in game.exe (+0.498 / +0.499 points on 84,877 / 85,124 weapon cases); 17 % of a 1-5 staff swing (I: arithmetic), less for bigger weapons | port only. Real build: a d2d test mode printing `player_blow`'s 256ths roll beside game.exe's FUN_0057b420 on the same cases (the skmiss `sk_missile_hit_test.cpp` pattern), oracle = this script's `dmg` part; the fix shape is `blow.frac += damage & 255` beside `phys` and the monster life in 256ths (the machinery of fix/missile-damage-fine) |
| D2 | Blunt weapon (ItemTypes 57 `blun`, which Staff, Scepter, Wand via `rod`, Club, Mace, Hammer reach) +50 % ED vs undead; stat 122 vs undead, 121 vs demons, 180 vs MonType (FUN_0057b660, FUN_0057b7d0, FUN_0057b6a0) | none | V run (FUN_0057b660, 60 / 60, the type test hooked), V read (121, 180), D (ItemTypes Equiv chain staf -> rod -> blun); that FUN_00629bb0 follows the Equiv chain is I (not run) | Act 1 zombies, skeletons, the Den: a staff, mace or scepter swing gets +50 points of damage % in game.exe (1.45x for a 10-Str Sorceress with a StrBonus-100 staff, I) | real build: a test mode calling `player_blow` with an undead target and a staff; oracle FUN_0057b660 + FUN_0057b420 (this script) |
| D3 | AR terms: 115 (defense 0 vs non-boss), 116 (-% defense, halved vs bosses), 123 / 124 (AR vs demon / undead), 179 (AR vs MonType), and stat 120 (+target defense) (FUN_0057d8a0, FUN_0057b7d0) | none (no stat ids defined, d2s_items.hpp:172) | V read | gear only; none on the starter gear | buildable: run FUN_0057d8a0 natively with the boss tests hooked |
| D4 | barehanded: p += strength; min = stat 21 or 1, max = stat 22 or 2 (jewelry flats replace 1-2, not added) | 1 + flats, 2 + flats, no Str term (combat.hpp:111-114) | V run | barehanded only (a character with no weapon); the bot never swings barehanded on purpose | port: 13,048 of 15,123 (seed 7) and 12,729 of 14,876 (seed 1) of the barehanded cases differ in whole points |
| D5 | AR' = AR + MulDiv(AR, %, 100) (C truncation of the share); no floor at 1 | `max(AR x (100 + %) / 100, 1)` | V run | only an AR' of 0 or less against a defense of 0 or 1 (game.exe 5 %, d2d up to 95 % with its AR floor of 1) and, rarely, a negative to-hit % (one case): 43 + 1 of 100,240 (seed 7), 36 of 100,240 (seed 1) | port |
| D6 | p floored at -90 once (skill ED + gear) | gear % floored at -90, then floored again with the skill ED | V read | only with gear % under -90 (none in practice) | not run |
| D7 (U) | FUN_0057b420 adds the player's stats 17 / 18 to p | weapon ED on the base only | U (Q3) | reading B: the weapon ED counts twice in game.exe | drill R1 decides; then the oracle's reading-B mode |

Not covered: swing speed / IAS / WSM (d2d `attack_speed` / `attack_ticks`, S upstream; the recon measured only the swing length), the defender's block (S monster-attack-trace.md: same rule, the monster side), leech, crushing blow, open wounds, elemental gear (S upstream combat.md), FUN_005d3ac0's state events, Smite / Zeal / Sacrifice dispatch (paladin-act1-recon.md row 2 asks for it; not traced here).

## Bot findings (described, not changed)

- B1 The resource simulator's melee is a flat 3 damage per 1.0 s swing that always lands (`internal/resource/sim.go:94`, applied at `internal/resource/sim.go:320`): game.exe rolls to-hit (for a clvl 1-8 Sorceress with about 75-100 AR against Act 1 defense, roughly 50 to 75 %, I) and a range with Str bonus and the 256ths tail. The policy's "melee is enough" choices are tuned on that.
- B2 `WeakPack` treats "every hostile at most clvl + 1" as "a weapon swing is enough" (`internal/resource/types.go:64`): the level ratio enters the to-hit (2 alvl / (alvl + dlvl)) but AR vs defense dominates; no AR or damage term.
- B3 The melee rung (`internal/resource/attack.go:74`) ignores the undead bonus: a Sorceress staff (Blunt by equivalence, D) gets +50 points of damage % against Act 1 undead in game.exe (about 1.4-1.5x), so the rung is better there than the simulator assumes; on d2d it is not (D2), so d2d under-rates the staff on undead.
- B4 Any d2d soak number for the early melee (paladin-act1-recon.md row 4, the clvl 1-11 Attack phase) carries D1 (minus about half a point a landed blow) and D2 (no +50 % vs undead with a staff, mace or scepter): d2d kills slower than game.exe in the first levels.

## PR candidates

None. D1, D2 and D4 are proven against a port of d2d's rule, not on a real d2d build with an existing differential harness (the standard of docs/process/upstream-pr-standard.md needs before > 0 / after 0 on the real build). The cheapest next step is a test mode on the upstream tree (the skmiss pattern) for D1 and D2.

## Drill recipe (lab binary or Game.exe, a throwaway character on one instance; not run)

- R1 (Q3, D7): equip a weapon with enhanced damage (any magic sword with ED), read the player's stats 17, 18, 21, 22 and the weapon's 21, 22 (the bot's memory reader, docs/d2-1.14d-memory.md stat list) and the char panel damage. Expect reading A: player 17 / 18 = 0 and player 21 = weapon base x (100 + ED) / 100 + flats. Reading B shows 17 / 18 = the ED.
- R2 (shield): a Paladin with a spiked shield (Armor.txt damage over 0): player stats 21 / 22 with and without the shield; expect no change (I).
- R3 (D2): on the fork's lab binary (docs/d2d-fork.md, docs/forkcheats.md), a staff Sorceress swings 50 times at a zombie and at a Fallen of the same level; mean damage per hit (life deltas in the fork's monster dump, docs/forkdbg.md) should be (100 + p + 50) / (100 + p) times higher on the zombie in game.exe, equal on d2d (p = Str x StrBonus / 100 + gear %).
- R4 (D1): 100 landed swings on one Fallen with a 1-5 staff: game.exe mean about 3.0 per hit, d2d about 2.5.

## Matrix cell updates (docs/research/sorc-skill-matrix.md section 3.12 and section 5 row 7; proposed text)

- 3.12 (a) rules: "do 1 FUN_0056f070: to-hit FUN_0057ec10 -> FUN_0057d9b0 (AR FUN_00622560 + MulDiv by mastery, 119, stat 325, 179), damage FUN_0057b7d0 -> FUN_0057b420 (stats 21 / 22 << 8 + 111, p = 25 + Str / Dex bonus + mastery + ED, + stats 17 / 18, roll in 256ths), Blunt +50 % vs undead (FUN_0057b660) [V run, sk-melee-trace.md]; a staff is Blunt by ItemTypes equivalence [D; the equivalence walk I]. Speed / IAS not traced [U]."
- 3.12 (b) d2d status: "same to-hit and whole-point range (100,000 cases, sk-melee-trace.md); floors the physical to whole points (D1, about -0.5 a hit), no undead / demon term (D2), barehanded rule differs (D4) [V run against a port]."
- Section 5 row 7: status "traced (SKMELEE, 2026-11-01): see sk-melee-trace.md; open Q3 / drill R1".

## Shared-file lines for the merger

- docs/decomp-queue.md, row 20, status column: `traced (SKMELEE 2026-11-01): docs/research/sk-melee-trace.md; to-hit and whole-point damage match d2d; d2d floors the physical (D1), lacks Blunt vs undead (D2) and the barehanded rule (D4); weapon ED at the player U (Q3, drill R1); no PR candidate`
- docs/BACKLOG.md (decomp section): `- skmelee (card [skmelee](../archive/openspec-index.md#skmelee), offline): the player's plain Attack traced (docs/research/sk-melee-trace.md, decomp row 20): FUN_00647bc0 is a skill's physical, not the blow (do 1 FUN_0056f070); to-hit and the whole-point weapon range equal d2d on 100,000 cases; d2d floors the physical to whole points (D1, about -0.5 a landed hit), has no Blunt +50 % vs undead (D2: staves, maces, scepters) and a different barehanded rule (D4); bot B1-B4; Q3 (weapon ED at the player) needs drill R1; next: a d2d test mode for D1 / D2.`
- docs/INDEX.md, decomp lane line, after the sk-nova entry: `, [research/sk-melee-trace.md](sk-melee-trace.md) (the player's plain Attack: to-hit, damage, Blunt vs undead, row 20)`
- CHANGELOG.md (Unreleased, docs): `- Decomp row 20 (SKMELEE): the player's plain Attack in game.exe 1.14d vs d2d (docs/research/sk-melee-trace.md, tools/emu/sk_melee.py).`
- docs/research/README.md: `- sk-melee-trace.md: the player's plain Attack (to-hit, damage build, Blunt vs undead, barehanded) in game.exe vs d2d; oracle tools/emu/sk_melee.py.`
- docs/research/sorc-skill-matrix.md: the three cells of the section above.
- Proposed glossary term (not written): **plain Attack** = skill 0, do 1 (FUN_0056f070): the weapon swing without a skill; **Blunt** = ItemTypes 57, reached by Staff / Scepter / Wand through `rod` and by Club / Mace / Hammer.
