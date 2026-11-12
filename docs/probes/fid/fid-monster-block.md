# fid-monster-block (decomp batch 7): the block chance of a monster that is hit, game.exe against d2d

> status: finished 2026-11-11 by agent decomp11 (Sonnet), offline (emulator on the worker host; no game, no live d2d, nothing pushed or posted, no fork branch). Result: **a NEW d2d deviation, V run on the game side: game.exe lets a monster block only when it is a "bit 4" row (Mephisto, Diablo, Baal's clone ...) or when it has a SHIELD component (skeletons always, the Fallen and corrupt rogues on some spawns), with the chance = its ToBlock stat; d2d lets every MonStats2 `mBL = 1` row block with ToBlock, always.** 60 of 733 rows differ at Normal (the first sample of each row also differs in 156 of 2,109 row x difficulty cases). In Act 1 Normal: d2d blocks a Fallen on 9 percent of landed blows, game.exe on about 1.5 percent on average (4 of 24 Fallen spawns carry the shield); zombies, goatmen, fetishes: neither blocks. Small in the early game, but the stat and gate are read, not only modelled. The d2d side is d2d's loaded data + the rule from its source (not a run of its fight code): the d2d claim is V run for the data and S read for the formula.

Tags: V run (game.exe's own code under the emulator), V read, D, S, I, U. Scripts: `tools/emu/bk_mon.py`, `bk_d2d_test.cpp`, `bk_run.sh`, `bk_cmp.sh`, `bk_rep.sh` (this card). Follows [smiteblock-trace.md](../../research/smiteblock-trace.md) (block bit and Smite), [monster-attack-trace.md](../../research/monster-attack-trace.md) (monster as attacker), [playerdef-trace.md](../../research/playerdef-trace.md) Q9 (player block chance).

## Q1. The chance FUN_00622720 gives a MONSTER defender (V read + V run)

`FUN_00622720(unit, scaled)` (read 0x622720..0x62281b). The unit type word `[unit]`: 0 player (shield needed, `(stat 20 + BlockFactor) x (dex - 15) / (clvl x 2)`: playerdef-trace Q9, out of scope here), **1 monster:**

1. Gate A: the table row of the unit's class (stride 0x1a8, tables +0xa78) byte +0xf AND the global at 0x6ce270 (value **4** when read after the game is booted in the emulator): non-zero opens the gate.
2. Gate B (when A is zero): `FUN_006225f0(unit)`: a class-range jump table for the hireling and special classes (0xf3 and up), else for a monster `FUN_00664860(unit, 7, idx)` with `idx` = the monster data byte +0xb: "the unit's component slot 7 (a shield) holds a choice at index idx" (V read of the head 0x664860..0x6648bf; the part after the first table lookup, FUN_006117d0, was not read: U).
3. Either gate open: the chance is **stat 20** (`FUN_00625480(unit, 0x14, 0)`, to-block), no dexterity or level term, the argument `scaled` is not used on this branch; cap **75** (0x62280e..0x622823, `cmp ebx, 0x4b`). Gate closed: 0.
4. FUN_0057dfb0 (read 0x57dfb0..0x57e01f): one roll `rand % 100 < chance` on the DEFENDER's seed after the hit roll passed; a third of the chance for a player in a moving mode (not reached for a monster).

Stat 20 of a monster comes from MonStats `ToBlock` of the game's difficulty (spawn, FUN_00573cb0; verify-skresist Q1 for the sibling stats).

## Q2. What it gives on real spawns (V run)

`bk_mon.py game` boots a game of each difficulty (`mm_launch.G`, the d9_revive.py setup: the real spawner FUN_00555230), spawns every MonStats row once and calls FUN_00622720 natively. `bk_mon.py rep` spawns each row 24 times (different spawn numbers) at Normal. `bk_d2d_test` prints d2d's REAL loaded `can_block` (MonStats2 `mBL`) and `ToBlock` per difficulty for every type. d2d's rule (S, ai.hpp:373 of the scratch tree): chance = `can_block ? to_block : 0`, no cap; the blow roll (`rng(100) < target.block`, combat.hpp:303) comes after the hit roll, as in game.exe.

| check | cases | result |
|---|---|---|
| game.exe stat 20 equals the ToBlock column of the difficulty | 2,109 row x difficulty cases | 24 differ: 8 names x 3 difficulties (overseer6, skeleton8, sk_archer11, skmage_fire7, skmage_pois7, vampire9, wraith9, willowisp8), each game value is the NEXT name's d2d value in this list: d2d's name lookup returns another row for them; the same rows verifycovstates found for boss column / Velocity (6 then, 8 here). Loader / lookup quirk of d2d for expansion-only names, tag **I** (not diagnosed; harmless at Normal Act 1) |
| game chance vs d2d chance, first spawn of every row x 3 difficulties | 2,109 | **156 differ** (d2d above the 75 cap: 0) |
| per-row over 24 spawns (Normal, 733 rows) | | see classes below |

Classes (d2d `can_block` and `ToBlock > 0` against the game's behaviour over 24 spawns):

| d2d | game.exe | rows | examples (Normal ToBlock) |
|---|---|---|---|
| no block, ToBlock 0 | never | 207 | gorgon, unraveler, chaoshorde, townspeople |
| **`mBL` 0, ToBlock > 0** | **always** | **5** | mephisto 20, baalcrab 40, baalclone 40, ubermephisto 20, uberbaal 40 |
| `mBL` 0, ToBlock > 0 | never | 442 | zombie 3 to 22, bighead, foulcrow ... (equal: both 0) |
| `mBL` 0 | sometimes | 1 | skeleton8 (21 of 24; the name-lookup quirk above) |
| `mBL` 1, ToBlock 0 | never | 2 | bear, ancientbarb1 (equal) |
| `mBL` 1, ToBlock > 0 | always | 24 | skeleton1-5 (10 to 30), pantherwoman, diablo 25, doomknight, diabloclone 50, slinger (equal) |
| **`mBL` 1, ToBlock > 0** | **never** | **36** | fallen2 16, the skmage_* and necromage mages (9 to 17), shadowwarrior 50, shadowmaster 50, eagle, wolf (pets) |
| **`mBL` 1, ToBlock > 0** | **sometimes (per spawn)** | **16** | fallen1 (4 of 24 spawns, ToBlock 9), fallen3 (14, 25), fallen4 (13, 36), fallen5-8, corruptrogue1-5 (16 to 21 of 24, 12 to 40), necroskeleton (19, 3), skeleton6 |

So the rows that differ from d2d (bold) are 5 + 36 + 16 = 57, plus the lookup quirk rows. In the per-spawn rows the Normal-difficulty block of a Fallen is on average `ToBlock x fraction`: fallen1 9 x 4/24 = 1.5 % (d2d 9 %), corruptrogue1 12 x 16/24 = 8 % (d2d 12 %). The shield choice is data byte +0xb (fallen1: 1 on the 4 shield spawns, 0 on the others; fallen2: 3 or 0 and still never blocks, so the gate tests more than "non-zero": FUN_006117d0's result, U).

What the rows tell about meaning: gate B is the SHIELD component of the visuals (a monster that is drawn with a shield blocks; skeleton, Fallen with a shield, corrupt rogue), gate A is a per-row flag that includes the bosses that block in the real game (Mephisto, Diablo, Baal's crab and clone; Andariel and Duriel do not). `mBL` in MonStats2 only says that the BLOCK animation mode exists. Tag: game side V run; the shield-component reading of gate B is I (the V read stops at FUN_006117d0).

## Q3. Does the order of the rolls match? (S, V run smiteblock)

Player blow on a monster: hit roll, then the monster's block roll on its own seed (FUN_0057ec10 -> FUN_0057dfb0), then dodge; smiteblock-trace V run (96,000 cases) showed the result bits; d2d `player_blow`: hit, block, then damage (combat.hpp:303). The stream (which random number each roll consumes) equals for a monster with `block > 0` on both sides; for a monster whose game chance is 0 (gate closed) game.exe still draws? **U**: FUN_0057dfb0 returns before the roll when the chance is not > 0 (read 0x57dfe2..0x57dfe4: `test ebx, ebx; jg`), so no draw; d2d draws only when `block > 0` (same).

## Deviations (d2d against game.exe)

| id | what | game.exe | d2d | tag | differential |
|---|---|---|---|---|---|
| D1 | which monsters can block | gate A (row bit 4) or gate B (shield component) | `mBL = 1` | game V run, d2d S read + loaded data V run | game side and data side built here (`bk_run.sh`); the d2d fight side (blow rolled against `Target.block`) not run; the fix needs a per-spawn shield flag |
| D2 | the chance on blockers | ToBlock stat, cap 75 | ToBlock, no cap | V run: no ToBlock above 75 exists in the tables (cap hit: 0) | none needed (unreachable) |
| D3 | per-spawn variation | the shield is chosen at spawn: Fallen 1 in 6 (fallen1) up to 2 in 3 (fallen3) | every Fallen the same | V run (24 spawns a row) | needs d2d's variant pick for layer 7 (`type_info` variants; I that d2d picks a variant at spawn) |
| D4 | bosses | Mephisto, Diablo, Baal crab / clone block with ToBlock | not (`mBL` 0 for Mephisto, Baal crab) | V run game side, data V run | with D1 |

Size for the bot (I): at Act 1 Normal the difference is at most 7.5 percentage points on a Fallen (9 vs 1.5) and 4 on a corrupt rogue; a blocked blow wastes one swing and shows the block animation. It does not move the kill-rate gap (the earlier monster-attack-trace found the monster rules explain about nothing of it); it matters for exact kill-time models of Fallen packs and for Act 1 Normal swords (Zeal, Smite).

## Bot findings (described, not changed)

B1 No bot code models monster block: grep of `internal/combat` for "block" finds only walk and line-of-sight blocking (`reach.go:138` BlockLOS, `fight.go:25`). Nothing to fix; a future kill-time model for the Paladin should use per-row block as game.exe's table above, not `mBL`. Consumer: any Paladin kill-time card (fid-pal-attack.md Zeal timing).
B2 (carried, monster-attack-trace B2, smiteblock) the bot's own block (a Paladin with a shield) uses the player formula; a running player blocks a third (`internal/d2dinput/run.go:25` run policy, equal in game.exe and d2d).

## PR candidate

None. The deviation is proven on the game side and in data only; the existing differential harnesses do not run d2d's blow against a monster with a given block. A fix in d2d needs the shield flag per spawned monster (D3) and a fight-side test: not built.

## Drill recipe (lab binary, docs/forkdbg.md; writing only)

K1 (D1/D3): `debug spawn fallen 40` on a field level (Normal), give each `debug mon set life 5000 maxlife 5000`, hit each with the same plain attack from a character with attack rating high enough to always hit; count blows that play the block animation (mode 9) or deal no damage: expect about 9 percent on every Fallen in d2d and about 1.5 percent overall on game.exe, clustered on the Fallen drawn with a shield. K2 (D4): Mephisto (`debug spawn mephisto`) at Normal: expect ToBlock 20 percent blocks of landed blows in game.exe, none in d2d. K3: a zombie pack: expect no blocks on either side.

## Matrix-cell update (proposed text)

New row proposal MA13 "monster defender block": "DEV | FUN_00622720 monster branch | game.exe: gate (row bit 4 or shield component) then ToBlock, cap 75; d2d: `mBL` then ToBlock; 57 of 733 rows differ at Normal (5 bosses d2d never blocks; 36 + 16 rows d2d blocks always where game.exe never / only with the shield), game side and data V run, fight side not run (decomp11, fid-monster-block.md) | C | 3 | S3 | P3".
