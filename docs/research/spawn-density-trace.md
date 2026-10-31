# Spawn density and champion/unique rolls: game.exe 1.14d traced vs d2d (agent DECOMPSPAWN, 2026-10-30)

> status: research, current. Builds on UPSTREAMGAPS item 3 (`docs/research/upstream-gaps/item3-monster-density.md`: d2d's generator equals game.exe per room, 20 seeds, 4 levels) and on the probes `docs/orig-spawns.md` / `docs/d2d-spawns.md`. Tags: V = read in game.exe (`disasm.py`) or run in game.exe's own code under unicorn by this agent (`tools/emu/spawn_bosses.py`); D = 1.14d patched MPQ tables read with `mpq-cat` (docs/process/mpq-data-reading.md recipe); S = upstream `docs/research/re/monsters.md`, `drlg.md`, `preset-specials.md` (psi29a/d2decomped e36922ea) or our probe docs; U = unverified. Classic 1.14d, Normal. Nothing was pushed, posted or filed; no game instance was started.

## Verdict in three lines

1. The 25 to 40 % density deficit is a MEASUREMENT artifact on both sides, bigger on the d2d side: game.exe only populates rooms within the player's near list, d2d (upstream and the lab fork, identical `gamedata.cpp`/`objgroups.cpp`) does the same, and our d2d sweep warped to a 30-tile grid (9 points on an 80x80 level) while the original census walked every frontier. The whole-level totals are the generator's (item 3): Blood Moor 97.8, Cold Plains 197.4, Stony Field 153.2, Dark Wood 218.8, Den 76.0 per seed; the original census reached 80 to 92 % of them, our d2d sweep 58 to 71 %.
2. The champion/rare gap in Cold Plains (original 0 rares, 0 champions in 4 visits; d2d about one event per seed) is the ROOM ORDER acting on one verified rule: a superunique counts toward the level's unique quota, and Cold Plains' quota is MonUMin 1 = MonUMax 1, so once Bishibosh's room is up no random champion or rare can roll. d2d has the same rule (`objgroups.cpp:657`, `++pop.uniques`); which comes first depends on the path, and 4 visits is too small to show a difference anyway.
3. No d2d bug was found in placement, density, group sizes, the unique/champion roll or the counting. No PR candidate (`docs/research/upstream-pr-candidates-spawn.md`).

## 1. When a room has monsters (bounds both censuses)

| Rule | Where | Tag |
|---|---|---|
| A room comes into play when the player's room changes: the new room's near list is walked three deep; depth 1 builds the room's tiles and room1 (FUN_0061b2d0 -> FUN_0061b190), depth 3 loads DT1s and preset units, depth 2 only marks | FUN_0061a110 -> FUN_0061b6f0 -> FUN_0061b490 / FUN_0061b390 | S (monsters.md "When a room populates") |
| The near list = the level's rooms under 6 tiles apart on both axes, plus flagged rooms of the levels next door | FUN_0066c370, FUN_0066be80 | S |
| Each tick, every room1 without flag 1 (+0x34) is populated from the head of the act's list (newest first): presets, stored units, then the density rolls, then the flag | FUN_0052d160 -> FUN_005559a0, FUN_00542b40, FUN_00552610, FUN_0054ec90 | S |
| A room is freed about 5 s after no player holds it within depth 1 and remade from storage on return (its monsters are stored, not re-rolled) | FUN_0052d240, FUN_005431f0 | S (drlg.md "A room's life on the server") |
| d2d: the same shape. `player_moved` brings up the arrived room and its near list (`close()` on room rects), `populate()` per room, rooms stay up for good | `components/game/gamedata.cpp:401`, `near_list` at `:263` | V (read; md5 of `gamedata.cpp`, `objgroups.cpp`, `monsters.hpp` equal in `~/d2d-lab/src` and `~/d2d-lab/fork/src`) |

So in both games a monster exists only if the player once stood in a room within 6 tiles of its room. The original census ("explore every frontier", coverage 91 to 99 % of the rectangle) therefore reads close to the whole level; a 30-tile warp grid cannot. Dead-end checked: the original's unit ids after a room is remade from storage are not re-read here (U); the census means (80 to 92 % of the generator) leave no room for systematic double counting.

## 2. Density and group sizes (the generator; d2d matches it, item 3)

| Rule | Where | Tag |
|---|---|---|
| Per room area: `(h x 5 / 3) x (w x 5 / 3)` rolls of the game seed, a hit when `seed % 100000 <= MonDen` (density capped at 10000); an 8x8-tile room is 13 x 13 = 169 rolls | FUN_0054ec90, FUN_0061ad50 | S + V (hit counts below) |
| MonDen 520 for levels 2, 3, 4, 5 and 600 for the Den, in all three difficulties; MonWndr 1; NumMon 3 | Levels.txt | D |
| So 0.88 hits per full room (169 x 521 / 100000); the trace gives 0.89 (Blood Moor 43.2 hits on 48.6 populable rooms per seed; Cold Plains 59.7 on 67.3; seeds 1..10) | `spawn_bosses.py` rolls per seed, region +0xc | V |
| A hit: a type by rarity (FUN_005bde80), then the unique/group roll (section 3), then a group: the leader, PartyMin..PartyMax minions (minion1) at radius 4, then `rand(MaxGrp - MinGrp + 1) + MinGrp - 1` more at radius 3; fallen1 and scarab1 (BaseId 19, 91) use 1..1 for the group | FUN_0054ec90, FUN_005b2830, FUN_0054ec40 | S |
| Group sizes (D, MonStats): zombie1 1-2, quillrat1 1-2, fallen1 1 + party 2-3, brute1 1, corruptrogue1 2-3, fallenshaman1 1 + party 2-6, cr_lancer1 1-2, skeleton1 1-3, zombie2 1-3, crownest1 1 (placespawn foulcrow1 2-4 when `rand(100) > 20`), goatman1 1-2, cr_archer1 1-2, fallen2 1 + party 2-4, corruptrogue2 2-4, quillrat2 1-3, cr_lancer2 1-3, sk_archer1 1-2 | MonStats.txt | D |

Analytic check (hits x mean group, rarity-weighted over the level's mon list; the region draws NumMon = 3 of the list's types, which the mean ignores, U how much that shifts it):

| Level | hits/seed (V, 10 seeds) | mean group (D) | analytic | generator mean (item 3, V, 20 seeds) | original census (S, 4 visits) | d2d sweep (S, 20 seeds) |
|---|---|---|---|---|---|---|
| Blood Moor (2) | 43.2 | 2.17 | 94 | 97.8 | 90.0 (92 %) | 69.0 (71 %) |
| Cold Plains (3) | 59.7 | 2.92 (+ about 8 for the unique's 3-6 minions and Bishibosh's 2) | 182 | 197.4 | 173.2 (88 %) | 127.0 (64 %) |
| Stony Field (4) | 52.2 on 58.0 rooms | 2.03 (+ about 16: 1.6 events x 4.5 minions, Rakanishu's 8) | 122 | 153.2 | 131.0 (86 %) | 90.5 (59 %) |
| Dark Wood (5) | 49.7 on 56.9 rooms | 2.56 (+ about 11) | 138 | 218.8 | 175.0 (80 %) | 125.8 (58 %) |

Hits per populable room are 0.87 to 0.90 on all four levels (V), the density rule exactly. The crude rarity-weighted mean group lands within 8 % of the generator on the two fallen1/zombie levels and 20 to 37 % under it on Stony Field and Dark Wood, where NumMon = 3 of 5 types, the rarity-1 types (brute1, goatman1, cr_archer1, cr_lancer2) are drawn less often and the big-party types (fallen2: 1 + 2..4, BaseId fallen1 so group 1..1, D) dominate the hits: the gap is the mean, not the rule (U how much each effect weighs; the generator's own per-seed totals are the V reference, so the analytic is a sanity check only). The measured columns are both lower bounds and the d2d one is the looser.

## 3. Unique, champion, rare: the roll and the counter (V, disassembly + trace)

| Address | What it does | Tag |
|---|---|---|
| `005be020` (ECX region, EDX room1) | reads region +0x2bd (MonUMax), +0x2c8 (uniques made) vs +0x2bc (MonUMin). If made < MonUMin and +0xc (rooms total) != 0: roll the room seed (+0x6c) `% 100`; if roll < `rooms_done(+4) x 100 / rooms_total(+0xc)` return 0. Else if made < MonUMax: roll `% 100`, if <= 5 return 0 (6 %). Else roll `% 100`: return 1 + (roll > 35). | V |
| `0054ee03` in FUN_0054ec90 | the caller: result 1 becomes 2; 0 goes to the unique path (FUN_005a43e0 at the 0x54eefe branch), 2 to a group | V |
| `0054ebc0` | per room: `++region +4` (rooms done) before any test; rooms total (+0xc) computed once (FUN_0061abf0) when still negative; then the room's no-population tests. Rooms done therefore passes rooms total near the end of a level (trace: 84..88 of 65..69 in Cold Plains), so the chance exceeds 100 % late | V |
| `005a0320` (EAX unit) | if the unit is a monster: find the level's region; `add [region + 0x2c8], 1` at `005a0362` unless monster data flag (+0x16) bit 8 is already set; then set bit 8. The flag byte values the original census saw (champion 0x0c, rare 0x08, Bishibosh 0x0a) are this bit 8 plus champion 0x04 or superunique 0x02 | V |
| `005a0b61` in FUN_005a09e0 | the random unique's placement counts it; `005a487b` (FUN_005a4850 with its count flag) and `005a48f4`, `005a496e` in FUN_005a48c0 count each champion of a pack | V (callers from the `--all` sweep) |
| `005a4a4e` in FUN_005a49b0 | the superunique spawn goes through FUN_005a09e0, so a superunique counts as one unique of its level (unless an act boss counted it first) | V |
| `005a0820`..`005a0830` in FUN_005a0760 | champion or unique: `rand(100) < MonUMod row 0 constants` (20, D) -> champion, else a rare with `rand(1) + 1 + difficulty` mods | V |
| MonUMin / MonUMax Normal: Blood Moor none, Cold Plains 1 / 1, Stony Field 1 / 2, Dark Wood 1 / 2, Den none; Nightmare 4 / 5, 4 / 5, 4 / 6, 4 / 6 | Levels.txt | D |
| SuperUniques: Bishibosh fallenshaman1 MinGrp = MaxGrp 2, AutoPos 1, mods 8, 9; Rakanishu fallen2 8, AutoPos 0, mods 17, 6; Treehead brute2 2, mods 5, 6; Corpsefire zombie1 5, mod 27; Stacks 0 (once a game) | SuperUniques.txt | D |

Trace of game.exe populating every room of Cold Plains in list order (`spawn_bosses.py 1-10 3`, V):

- Every seed had exactly one counted random event or none: the MonUMin roll succeeded at rooms 5 to 17 of 65 to 69 in 9 of 10 seeds (a rare cr_lancer1 / corruptrogue1 / brute1 / fallenshaman1, or in seed 6 a champion pack counted 3 times), then Bishibosh (class 58, counted at `005a0362` after the preset spawn) raised the count to 2 (4 in seed 6) and no further roll could succeed (MonUMax 1).
- Seed 3: Bishibosh's room came first in list order, count 1 = MonUMin = MonUMax, and NO random champion or rare was rolled in the whole level (57 hits, all groups).
- Blood Moor (`1-10 2`): no roll ever succeeds (MonUMin = MonUMax = 0), 32 to 54 hits per seed.
- Stony Field (`1-10 4`, MonUMin 1 / MonUMax 2): 1.6 successful rolls per seed (1, 2, 1, 2, 2, 3, 2, 1, 1, 1), counts 2 to 6 (events plus Rakanishu); Dark Wood (`1-10 5`): 1.7 per seed (2, 1, 1, 2, 2, 1, 2, 2, 2, 2). The probes: d2d sweep 27 events / 20 seeds = 1.35 on both levels (partial coverage), original 7 / 4 = 1.75 in Stony Field and 2 / 2 = 1.0 in Dark Wood: all within sample noise of each other and of the generator; the second event is the 6 % roll, so it needs the first one early.

What this says about the probes: the random event of Cold Plains is one per level at most and it dies the moment Bishibosh's room is up. In the original census (walk order, Bishibosh 4/4) it is enough that his camp came up before the rising roll hit: the roll's chance is `rooms done / rooms total`, so with the explorer entering from the Blood Moor and the first rooms at 5 to 15 % chance per hit, 0 events in 4 walks is unremarkable (and with n = 4 nothing in Cold Plains or Dark Wood champion counts is a measurable difference: Stony Field 3 champion packs in 4 visits vs d2d 7 in 20 seeds has p about 0.08 under d2d's rate). The d2d sweep (grid order) got its event before Bishibosh in 11 of 11 seeds where he was seen, which the same rule allows. The champion share of counted events in d2d (6/20, 7/27, 6/27 for levels 3, 4, 5) fits the 20 % roll; the original's 3 of 9 fits too (p about 0.26).

## 4. Superuniques

- Act 1 superuniques are preset units of their DS1 (AutoPos 1: at the marker; Rakanishu AutoPos 0: a random spot), made when their room is populated, once a game (Stacks 0), through FUN_005a49b0 which gives Mod1..3, `difficulty` more unique mods and MinGrp..MaxGrp minions at radius 3 (S, preset-specials.md 1.4; code `objgroups.cpp:629`). Whether the DS1 holding the camp is always picked for the level is not traced here (U); the original showed Bishibosh, Rakanishu and Treehead in every visit (4/4, 4/4, 2/2, S).
- d2d's 11/20, 12/20, 10/20 for the same three: the sweep's grid never brought their rooms up (section 5); not a spawn rule. Corpsefire 0/4 in the original vs 20/20 d2d: the Den explorer declared "nothing left" after 66 to 72 s without his room in the near list (S, the probe's own reading); the rule above gives no reason for him to be absent (U until walked by hand).
- Bishibosh's 2 minions match the original's 2.0 minions per Cold Plains visit (S).

## 5. Why our d2d sweep reads 58 to 71 %, and the apples-to-apples totals

- `cmd/d2dspawn/main.go` `sweep()`: `const gap = 30.0` in level tiles, grid points at 15, 45, 75 on an 80x80 level = 9 warps, 12 ticks each, `monsters` read after each. Each warp brings up the arrival room and rooms whose rects are `close()` (6 tiles) to it: about a 3x3 block of 8-tile rooms per point, so rooms more than about 14 tiles from every grid point never populate. Nine 20-tile blocks on an 80-tile side cover about 2/3 of each axis; the measured 58 to 71 % is that (S for the numbers, V for the code path).
- Fix (not applied: cmd/d2dspawn is not this card's): `gap = 8` (one room; 100 warps at 12 ticks is about 20 s per level) or walk the level; then the sweep must reproduce the generator's per-seed totals (seed 1 Blood Moor 95, item 3) and the original census should sit at 80 to 92 % of the same columns.
- Comparable table today: original 90 / 173 / 131 / 175 / 56 vs the generator 97.8 / 197.4 / 153.2 / 218.8 / 76.0 (both V-side numbers are per seed means; the original's seeds are unknown, so only means compare). d2d itself is not in deficit: it IS the generator (item 3: 577/787/710/703 rooms match; my spot check `diff_drlg.py 7-7 3 monsters`: 41/41 rooms, from `~/d2d-lab/pr/gaps` with its built `drlg-dump`, V).

## 6. Open points

- The near list's "6 tiles" and the depth rules are S (upstream's reading), not re-read here.
- The original's walk order vs Bishibosh's room: to turn section 3's explanation from "allowed by the rule" into "observed", log the flag-0x0a unit's first-seen time against the first flag-0x08 unit's (a later probe; ORIGPROBES' flagged-unit log is not in the repo, `docs/orig-spawns.md` names it).
- The DS1 choice that places the three camps (always present?) and Corpsefire's room in the Den: U.
