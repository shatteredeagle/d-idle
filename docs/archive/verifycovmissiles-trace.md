# Independent verification: the missiles coverage first pass (covmissiles-trace.md)

> status: archived 2026-11-11 (TASK-311, docs/process/coordinator-efficiency.md rule 8): finished one-off report, no inbound link when archived; stub left at docs/research/verifycovmissiles-trace.md.

> status: card verifycovmissiles, 2026-11-02, Sonnet 5.5, offline (emulator and a hand-built d2d probe on the worker host; no game, no d2d slot, nothing pushed or posted). Subject: docs/research/covmissiles-trace.md on branch agent/covmissiles (not yet on main when this was written) and its two ledger lines. Method: read game.exe first, write my own model, run game.exe's real code with my own harness and seeds, only then compare with the author's numbers. None of the author's scripts was run; two of upstream's (`missiles.py` Game class, `sk_missile_hit.py` / `skill_damage.py`, both already verified by earlier cards) are the emulator base, the cases, skill table and every comparison are mine.

Tags: **V run** (own harness, counts and seeds given), **V read** (own disassembly read, addresses given; d2d read at upstream main), **D** (patched 1.14d tables through `mpq-table.sh`), **S** (an upstream or earlier doc), **I** inferred, **U** unverified.

## Verdicts in short

| claim of covmissiles-trace.md | verdict | main limit |
|---|---|---|
| (B) game.exe's record equals the table; `FUN_0059fa30` makes range, velocity, MaxVel, Accel as the formula says; d2d loader and rules agree | **VERIFIED** (48,624 own cases, 0 differ; 302 rows x 14 columns loader vs table, 0 differ) | I re-checked 6 of the 17 record columns against the table (the flight ones); the other 11 offsets I only read in `FUN_005adf10`; d2d's production range code does not call the helper that was compared (below) |
| (A) hit roll D1 on eight more skills (41, 46, 47, 51, 55, 57, 60, 121) | **VERIFIED** (77,560 own cases, d2d before 21,650 differ, after 0, per skill >= 500 differ before, 0 after) | the roll is run on a synthetic missile that carries the skill's pair; the real missile's own hit function was read (addresses below) but not run for each row |
| ledger line, six hit rows (60 62 96 166 233 264, 103,439 cases) | **EARNED, with that limit** | row 233: hit function 22 rolls on its own (0x5ade7d), a different call site of the same roll |
| ledger line, 222 table / flight rows (4,440 cases) | **EARNED for loader and flight values** | the range comparison is of `rules::missile_range`, which d2d's production code does not call (it computes `range + lev_range x lvl` inline); the note has a stray token "SKDRUID" at its end |
| Blaze 67 / Fire Wall 69 left out because of `pSrvDmgFunc` 3 | **conservative, now resolved by reading**: that function does not touch the pair (below) | the hit roll of those rows could be claimed; not done here (ledger id overlap, see Ledger) |
| D1 (SKMISS roll top) | **VERIFIED** again, no new deviation id needed | none |
| D2 (d2d loads 302 of 684 rows) | **VERIFIED** (own run of the real loader: 302; the names in the author's list are not loaded) | whether another d2d rule carries those skills stays U |
| D3 (CollideFriend, CollideType unread) | **HOLDS WITH LIMITS** | d2d side VERIFIED (0 matches in `components/`); game side: CollideFriend read in `FUN_005adf10` (V read); the author's list of rows is incomplete (25 rows carry it, D) |
| D4 (pierce rule not read on the game side) | **now read AND run, and it is a larger deviation than the author knew**: see D1 below | |
| B1, B2, B3 (bot) | **B1 VERIFIED, B2 HOLDS (flight rule S), B3 HOLDS WITH LIMITS** (12 or 13, not 11, Sorceress / Paladin skills keep the missile in `srvmissilea`) | |

## What ran

| run | command (worker host, repo tools/emu) | cases | seed | result |
|---|---|---:|---|---|
| flight, own grid | `run-on-worker.sh vcm_run.sh 4000 11` | 16,312 = 684 rows x 18 levels (0 1 2 6 11 13 18 24 31 37 45 63 80 99 127 128 200 255) + 4,000 random, 2,004 with a perturbed record | 11 | made 16,312 (none refused), game vs my model of the record **as loaded in game memory** 0 mismatches; real d2d `missile_velocity` / `missile_range` vs game 0 / 0 |
| flight, second | `vcm_run.sh 20000 5` | 32,312 (10,088 perturbed) | 5 | the same, 0 / 0 / 0 |
| record vs table | `vcm_rowcmp.py` on run 11 | 684 rows | none | Range, LevRange, Vel, VelLev, MaxVel, Accel read back from game.exe equal the patched Missiles.txt on all 684 rows (0 disagreements) |
| d2d loader | `vcm_loaded.sh` (own program, the real `load_game_data`) | 302 rows x 14 columns | none | d2d loads 302 of 684 rows; 0 column differences against the table; every id of the author's two ledger lines (228) is loaded |
| hit roll, own cases | `vcm_hit_run.sh run 3000 101` | 20,280 | 101 | before 4,915 differ, after 0 |
| hit roll, second | `vcm_hit_run.sh run 40000 202` | 57,280 | 202 | before 16,735 differ, after 0 |
| pierce at creation | `vcm_pierce.py` | 1,616 = 4 rows x 4 owner states x 101 chances | none | `FUN_0059f940` equals my model on all (below) |

Own choices that differ from the author's: the flight record is perturbed (Range, LevRange, Vel, VelLev up to 255, MaxVel, Accel written into game memory, a negative LevRange, levels to 255), so the C-division and the clamp are exercised, and the model reads the record from game memory, not from the table text; the hit cases use as synergy partners **every** skill id 36..130 at random levels (94 partners, not a window of 25 ids), 12 levels per skill up to 60, resistance -100..120, and the d2d side is the real `rules::missile_damage`, `missile_blow` and `hurt()` (upstream's `sk_missile_hit_test.cpp`, built before and after the SKMISS fix 23e2baec with `vcm_hit_run.sh build`). The etype of the eight skills is read from game.exe's own table (1 fire, 2 lightning, 4 cold) before every run.

Per skill (runs 101 + 202), cases / differ before the fix / after: 41 Inferno 9,570 / 4,586 / 0; 46 Blaze 9,726 / 5,229 / 0; 47 Fire Ball 9,720 / 1,192 / 0; 51 Fire Wall 9,810 / 4,430 / 0; 55 Glacial Spike 9,642 / 1,117 / 0; 57 Thunder Storm 9,645 / 1,787 / 0; 60 Chilling Armor 9,742 / 2,803 / 0; 121 Fist of the Heavens 9,705 / 506 / 0. Totals 77,560 / 21,650 / 0. The diff lines are `id min max life`, so min and max of the pair (synergies, mastery) agree on all of them too.

## Own reading of game.exe (V read; `vcm_dis.sh`, `vcm_all.sh`, `vcm_tbl.py`)

- **R1 the hit, `FUN_005adf10` (ECX game, EDX missile, arg1 unit hit, arg2 `ended`)**: row = class at `[0x744304]+0xb64`, 0x1a4 a row. Order: a NextHit row (+0x188) skips a unit with state 0x56; `FUN_0064a470` / `FUN_005ada40` eligibility; **hostile test `FUN_00554200`, else it ends the call unless CollideFriend (+0x187)** (0x5adfdc..0x5adfea); CollideKill (+0x186) sets the end flag (0x5ae034..0x5ae040); ToHit (+0x18b) calls `FUN_0057d9b0`; **the row's pSrvHitFunc (word +0xe, table 0x73c840) runs, and if its result has bit 2, `FUN_005a89a0` rolls the record at 0x5ae150, then pSrvDmgFunc (word +0x10, table 0x73c960) runs on the record, then `FUN_005adcd0` applies it**. The end flag then runs `FUN_0064eba0`. V read.
- **R2 the roll `FUN_005a89a0` has 8 callers**: 0x5a9d31 (hit function 1, Fire Ball), 0x5a9f47 (2), 0x5aacb2 (14), 0x5ac604 (47), 0x5acd33 (56), 0x5ade7d (22, Fist of the Heavens delay: it picks its unit and rolls itself), 0x5ae150 (R1) and 0x5afb0f (do function 27). The tables (`vcm_tbl.py`, read from the loaded image): hit functions 71, damage functions 31, do functions 389 entries. V read + V run.
- **R3 the do functions reach R1**: `FUN_005ae1f0` (the per-frame move; do function 1, table entry 0x5b0bc0, is a bare `jmp 0x5ae1f0`) calls `FUN_005adf10` at 0x5ae231, 0x5ae25f, 0x5ae289, 0x5ae3d5, 0x5ae3ea. Other do functions call it (31 call sites, for example do 2 at 0x5ae475, do 5 = Blaze and Fire Wall at 0x5ae55d) or tail-jump to it (do 3 = Thunder Storm rows 166-169: both of its branches end in `jmp 0x5ae1f0`, at 0x5ae49d and 0x5ae510, the second after `FUN_0064cb90` places the unit in its room). So the generic roll at 0x5ae150 is what these per-frame missile hits meet. V read (grep of the dump plus the three function bodies); I did not read every one of the other do functions, so "every" is not claimed.
- **R4 pSrvDmgFunc 3 (`FUN_005ad220`, rows 67 Blaze, 69 Fire Wall, and four others)**: one seed step on the missile (`lo x 0x6AC690C5 + hi`), and if `step & 0x7f` is below the row byte +0x78 it ORs 0x4000 into the record's word +4. It does not change min, max or the rolled value, and it draws AFTER the roll. So the author's dead end (the roll of 67 / 69 unread) is answered: same roll, the hit-record flag 0x4000 is the only extra (what it does: U; what column +0x78 is: U). V read.
- **R5 velocity and range**: not read line by line by me; my number is the run of the real function on 48,624 cases (including a record I wrote myself), which is the oracle, not a re-derivation: `((Vel + VelLev x lvl / 8, C division) << 8) x 75 / 100`, `Range + LevRange x lvl` clamped to a short, `MaxVel << 8`, `Accel`. VelLev read as an unsigned byte (values 128..255 in my perturbed cases match the unsigned model). V run.
- **R6 pierce at creation, `FUN_0059f940` (EAX owner, arg1 missile; its one caller is the maker at 0x59fe72)**: owner type 0 or 1, missile type 3, the row's flag byte (+4 & the byte at 0x6ce270, which is 4) set; chance = `stat(owner, 0xa6)` + `stat.mod(owner, 0x9c)`; if it is 0 nothing happens. Else the seed is `{stat.base(owner, 0x148), 0x29a}` (`FUN_00650e40`), and up to **four** draws: `lo' = lo x 0x6AC690C5 + hi`, `lo' % 100 < chance` counts one and goes on, else stops; the missile's stat 0x148 := the count (`FUN_00627260`). The hit (`FUN_005ada80`, called from R1 at 0x5ae027 when the owner has stat 0xa6 or 0x9c) returns 2 and lowers the missile's stat 0x148 by one when it is nonzero, 3 otherwise; bit 1 of that result is what lets CollideKill end the missile (0x5ae19f). V read; the creation function is also V run (D1 below).
- **R7 names (D, `itemstatcost`)**: 156 `item_pierce`, 166 `skill_pierce` (the Amazon passive Pierce, Skills.txt id 33, `passivestat1`), 328 `pierce_idx`. The owner's `pierce_idx` is raised by one at 0x5980a6 (a unit-update routine; what triggers it: U).

## Claims, one by one

| claim | verdict | evidence |
|---|---|---|
| Q1 record layout, 17 columns, 0 disagreements on 684 rows | **HOLDS WITH LIMITS** | own run: the 6 flight columns equal the table on 684 rows; the other offsets (+0x186 +0x187 +0x188 +0x18b +0x18c, +0xe, +0x10) agree with how `FUN_005adf10` uses them (V read); +0x183 CollideType, +0x185, +0x18a, +0x18d not touched by me. The Pierce column is flag bit 4 of byte +4: the flagged rows in game memory are exactly the 60 rows with Pierce in the table (V run) |
| Q2 range, velocity, MaxVel, Accel, 13,680 cases | **VERIFIED** (48,624 cases) | `vcm_run.sh`; none refused |
| Q3 speed Vel x 0.046875 subtiles a frame | **HOLDS** (arithmetic on the V values: Vel x 192 x 16 / 65536; step rule S) | not rerun |
| Q4 Accel on 21 rows, VelLev 4, LevRange 32 | **VERIFIED** (D, counted from the table; the Accel rows are the author's list; nova and frost nova -1000, Blessed Hammer and Blade Shield +250 up to 30, Baal inferno +250 up to 40) | |
| Q5 same chain as SKMISS for 8 more skills | **VERIFIED for the roll and apply** (77,560 cases); **HOLDS WITH LIMITS for the path** | R1..R3: the generic hit calls the same roll; Fire Ball's hit function 1 and the Fist of the Heavens hit function 22 also roll (R2); rows with pSrvDmgFunc 3 follow the same roll (R4). Glacial Spike's hit function 13 (0x5aa8b0) contains no call of the roll |
| Q6 columns d2d never reads | **VERIFIED** (grep of `components/` at upstream main: 0 matches for CollideFriend, CollideType, ClientCol, ClientSend, KnockBack, SoftHit, CanSlow, NoMultiShot, CanDestroy, LastCollide, Holy, Town, Qty, SrcTown) | |
| Q7 302 rows | **VERIFIED** (V run, real loader) | the names of the author's D2 list that are NOT loaded: infernoflame2 / 3, thunderstorm2-4, blessedhammerex, sanctuarybolt, sanctuarycenter, redemption, redemptionfail, blizzardexplode1-3, frozenorbexplode, meteor, meteorexplode, meteortail (V run); loaded: infernoflame1, thunderstorm1, fireball, glacialspike, chillingarmorbolt, blessedhammer, fistoftheheavensdelay and bolt, firewall, blaze, meteorcenter, holybolt |
| Q8 d2d CollideKill / Pierce rule | **VERIFIED as a read of d2d, location differs** | `collide_kill && !(pierce && rng(100) < psum[kPierceIndex])` is at `fight.cpp:2971` in upstream main; the author's `fight/missiles.cpp:632` is another checkout's path (not checked); game side now read: D1 below |
| ledger "6 hit rows" and "222 rows" | see Ledger | |

No claim of the first pass is "no deviation by re-derivation of d2d's formula": both flight sides (game and the real d2d rules / loader) ran natively in the author's run and in mine. The hit "after 0" is the real d2d function against game.exe's `FUN_005a89a0` + `FUN_0057c6c0`. The weak spot is different: the hit harness compares the **roll and apply** on a synthetic missile carrying the skill's pair, not the whole real missile (collision, sub-missiles, hit functions), so a "0 after" says the damage a hit deals is equal, not that d2d hits at the same moments.

## Deviations d2d vs game.exe (new from this card; the author's D1..D4 keep their numbering in covmissiles-trace.md)

| D | rule in game.exe | rule in d2d | tag | differential |
|---|---|---|---|---|
| D1 | **pierce**: the chance is stat 166 (skill_pierce) + stat 156 (item_pierce), rolled ONCE per missile at creation as up to four consecutive draws on a seed `{pierce_idx, 666}` (the owner's stat 328, +1 each time the unit-update routine at 0x5980a6 runs), so a missile can pierce 0..4 units and then stops; only the 60 rows with the Pierce column | `collide_kill && !(pierce && rng(100) < psum[328])` per unit struck: stat 328 is read as the chance, no cap, a fresh roll per hit. d2d never names stats 166 or 156 (`vcm_grep.sh`: 0 matches) and the comment at `d2s_items.hpp:190` calls 328 "chance to pierce" | game: **V run** (`FUN_0059f940` native, 1,616 cases = rows arrow, diablight, strafebolt, javelin x owner stat-328 base 0, 1, 7, 12345 x chance 0..100: 0 mismatches against the model; a row without the flag keeps count 0; with base 0 the draws are 66, 18, 35, 30, so chance <= 66 gives 0 pierces and >= 67 gives 4); d2d: **V read**; effect in d2d (an Amazon with Pierce does not pierce, because psum[328] is 0 unless an item carries stat 328, and 166 is stored under 166): **I**, U until run | none exists. Build: a d2d probe that sets `psum[166]` to a chance and calls `Fight`'s missile-end test for a pierce row over seeds, oracle = game's `FUN_0059f940` count and `FUN_005ada80` spend (both runnable as in `vcm_pierce.py`, with the owner's stat 328 stepping 0, 1, 2 ...) and the number of units a missile passes; the frame of a real arrow through a line of monsters |
| D2 | `Range + LevRange x lvl` unclamped by the table, clamped to a short by game.exe | `rules::missile_range` clamps like the game, but **production code does not call it**: `fight.cpp:1050, 1263, 1273, 1429, 2614, 2688, 2745, 2792, 3018`, `ai.cpp:1646, 2189`, `ai.hpp:634` compute `range + lev_range x lvl` inline (some with `max(.., 1)`) | **V run** (helper equals game on 48,624 cases), **V read** (the inline code) | only matters past level 327 (LevRange 100 at most, D); no deviation reachable, recorded so that nobody reads the 6,040-case "0 differ" as a test of the production path |
| D3 | CollideFriend rows: **25** (D: holybolt 55, shout 149, fistoftheheavensbolt 234, battlecommand 236, battleorders 237, cairnstones 288 / 289, radamentholybolt 349, healingbolt 357, izual lightning 382 / 383, bomb explosion 387, impfireballexplode 405, catapultchargedballbolt 408, imp spawn monsters 409, catapult spike on ground 413 / explosion 414, healing vortex 425, sucshockfield 446, vine beast rows 466-470, sentryspikeonground 497); read in R1 (a non-hostile unit is still hit) | column unread (Q6) | game V read, d2d V read | as the author's D3 |

## Bot findings

| B | where | verdict |
|---|---|---|
| B1 (author) | `internal/data/extra.go:361-366` `RangeAt` uses `lvl - 1` | **VERIFIED**: game makes `Range + LevRange x lvl` (my 48,624 cases use lvl as given); `data_test.go:371` pins the `lvl - 1` form on a fixture row |
| B2 (author) | `internal/combat/threat.go:15-36` speed ignores Accel / MaxVel / VelLev | **HOLDS**: the rows are as listed (D); the in-flight deceleration is upstream's proof (S), not rerun |
| B3 (author) | `internal/combat/classdata.go:183-193` | **HOLDS WITH LIMITS**: `range` is `none` on 54 of the 60 Sorceress / Paladin rows, `h2h` on 5, `rng` only on Inferno (D); 13 of them (12 without Inferno), not 11, keep the missile in `srvmissilea` |
| B4 (new) | the bot has no pierce model; none needed until D1 is confirmed live: an Amazon's arrow count through a monster line differs between d2d and the game. | I (not searched beyond a read of threat.go / classdata.go) |

## Ledger

- The author's two lines are earned for what they say (the numbers 103,439 and 4,440 re-add; before-fix 40,850 re-adds; the percentage 9 to 42 re-adds: 288 of 684 run-tested). Suggested edits, not made here (the lines are not on main yet, and `coverage.py` refuses an id in two lines): on the 222-row line, remove the stray "; SKDRUID" at the end of the note, and add "range: the helper, not the production inline code"; on the hit line add "roll and apply on a synthetic missile".
- Not changed: ledger.tsv and rule-coverage.md (`coverage.py --check` passes). Ids I could add once the author's lines are on main: 67 and 69 (`TESTED-DEVIATION`, 19,536 own cases, before 9,659 differ, after 0; the id 69 sits in a READ-ONLY line today and 67 in the author's 222-row line, so the split is the merger's); the 60 Pierce rows stay READ-ONLY (no d2d run).

## PR candidates

None. D1 (pierce) is proven on the game side only; no differential exists on the d2d side, so the rule "mismatches before > 0, after the fix 0" cannot be met. The SKMISS roll fix is unchanged (23e2baec; my 77,560 cases are more evidence for it).

## Dead ends and limits

- The real missile path per skill (collision, the row's own hit function) was not run, only read; a run needs a player, a monster and the stat hooks `life_apply.py` holds, one harness per row class.
- What triggers the owner's `pierce_idx` increment (0x5980a6) and how often is open (U); the live distribution of the pierce count therefore is too (the draws for index 1, 7 and 12345 are 51 31 12 93, 69 95 50 82, 87 64 71 25, so it is not a step function once the index moves).
- The land-range of hit function 36 (`fight.cpp:2728-2731`, `missile_velocity(.., 1)`, level fixed at 1) against game.exe's flag-0x400 range rule (`(distance << 16) / (velocity << 4)`): not run (U); matters only for VelLev rows (spike1-4).
- The harness cannot test a magic (Holy Bolt) row: d2d's test switches fire, lightning, cold (etype 0..2) and skips the rest silently; a missing line would show as a diff, so the zero after is real for the eight skills.
- The author's remark that `labrun` rejects `|` in arguments is right for scripts; I used one grep per call or a script.

## Drill recipe for the fork verbs (lab only; nothing here was run in a game)

- P1 (D1 pierce, live): a throwaway Amazon with Pierce at level 20 (stat 166 = the skill's percent, `dump stats`), spawn a row of monsters in a line 1 subtile apart (`debug spawn`), `debug cast` Fire Arrow / Magic Arrow (rows with Pierce, D) along the line 30 times, `debug missiles` per frame: count the units each arrow strikes before it is gone. game.exe: 0 to 4, never more; d2d: unbounded and (if psum[328] is 0) always 1. Repeat with the Pierce skill at 1 and 20.
- P2 (D2 range): `debug cast` a LevRange row (blaze, firewallmaker) at levels 1, 10, 20 and read the frame count: Range + LevRange x level.
- P3 (Q5 limit): `debug cast` Fire Ball / Glacial Spike / Chilling Armor at a monster of 0 resistance with `debug mon set` life large, 40 casts, compare the life drop per hit with the table pair (the top of the pair must never be dealt).

## Matrix-cell updates (proposed; docs/research/sorc-skill-matrix.md and the Amazon matrix are not edited here)

- Sorceress rows 41, 46, 47, 51, 55, 57, 60 (and Paladin 121): "hit roll D1 independently reproduced (verifycovmissiles, about 9.7k cases per skill, 0 differ after the SKMISS fix; synergy partners all of 36..130)". Rows 46 and 51: replace "pSrvDmgFunc 3 not read (U)" by "pSrvDmgFunc 3 (`FUN_005ad220`) only sets a record flag after the roll (V read)".
- Amazon rows with a Pierce row (Fire / Magic / Cold / Exploding / Freezing arrow, Lightning Fury, javelins): "pierce differs: chance 166 + 156, at most 4 pierces seeded by pierce_idx, vs d2d stat 328 percent per hit (verifycovmissiles D1, V run game side, d2d read)".

## Shared-file lines for the merger

See `openspec/changes/verifycovmissiles/merge-lines.txt`.
