# fid-boss-mods (TASK-85, matrix row MA06): unique, champion and superunique modifiers, d2d against game.exe

> status: finished 2026-11-08 by agent decomp6 (Sonnet). Result: **pr-mods is MERGED upstream (#157 and the follow-ups) and its level / life / defence / experience / damage / to-hit / speed claims hold: 0 real differences in 184,064 oracle cases; one open deviation of the earlier trace is reproduced and has a fix**: a Cold Enchanted unique's chill is one frame on d2d (its cold carries no length), game.exe adds level x 5 + 100 frames (trace E2). Local PR candidate `fid/cold-enchanted-length` on the fork remote (not posted); the other remaining rows (two enchants on one unique, minions carrying no mods) are described, not fixed. Open (U): the 49 cases at the very top levels, game.exe's reading of "stat 12 at that moment" there.

Tags: V run (game.exe's own code run by the emulator and compared with d2d's real or copied code), V read, D, I, U. Worker host, scratch tree = upstream main a5b970d5 (the boss_level / umod runs use the TASK-83 candidate's tree, which does not touch monsters).

## What was upstream already (checked first)

The log of upstream main: #157 (boss life / defence / experience from the MonLvl row of the MonStats level, minions' leveladd, ghostly speed, berserker leveladd; `boss_level.py` + `test_boss_level`), a boss's death (MonUMod 22) oracle `boss_deaths.py` and Blood Raven's row (#258, #260, 2026-11-08), `test_boss_resists` (at most two immunities, stone skin before the gate; trace S1), "a zero-life roll lives, any hit kills it" (5ca89a60). `pr-mods-proof.md` is merged; do not open it.

## Q1. pr-mods evidence on current main (AC #1)

| Run | Cases | Result |
|---|---|---|
| `boss_level.py gen 100000 20261108` (every MonStats row x difficulty x kind c / u / s / m / p, then 100,000 random): game.exe's real FUN_00573cb0, FUN_005a2120 and FUN_005a4f50 against d2d's real `make_boss` (`test_boss_level`): level, life, defence, experience, to-hit, A1 damage, speed | 136,624 | 5,973 differ, **all** "game.exe life 0 to 255 in 256ths, d2d 256" (`sk6_boss_filter.py`: any other difference **0**) |
| `boss_level.py gen-area 30000 5` (Nightmare / Hell start at the area's MonLvl2Ex / MonLvl3Ex) | 47,440 | 1,604 differ, all the same life-under-one-point case; any other difference **0** |
| `boss_deaths.py` (MonUMod 22 by class, FUN_005dfbf0's area, mod 21's FUN_005a3aa0) | | ok |

The 5,973 / 1,604 are the representation upstream chose on 2026-11-08 (5ca89a60: `Monster::lifeless` holds one point; game.exe's life is a roll that came to under a point and "any hit kills it": a landed hit leaves under 256ths, so both die from any damage that lands; they differ only for a hit of 0 damage). The harness `boss_level.py compare` has not been taught that yet; as the mismatch list is empty apart from it, I rate the claim **equal** and tag the two harness lines I (the comparison of a sub-point life with a lifeless unit is by reading 5ca89a60, not run).

Then the 157,618-case spawn-stat differential of the trace (`umod_stats.py 100000 1`, `test_umod_stats` = d2d's `boss_stats` applied as `make_boss` does), fields level, exp, life, damage %, to-hit %, speed %, six resists, defence, elemental min / max, cold length, mana burn:

| kind | single mod | several mods |
|---|---|---|
| champion (c) | 0 differ in every field | 0 |
| unique (u) | 0 differ except cold_len (E2 below): 3,333 | elemental min / max 295 + 285 + 248 (E1: two enchants on one unique), cold_len 2,290 |
| superunique (s) | 0 except cold_len 499 | elemental 1,625 + 1,569 + 1,634, cold_len 7,161 |
| minion (m) | the harness gives minions no mods (as d2d tags them): damage / to-hit 3,303, speed 2,928, enchanted damage ~1,900 each element, mana burn 1,867, cold_len 3,373 | similar |

The trace's levels, life, defence, experience, AR, damage and speed rows are therefore equal for c / u / s; the remaining differences are exactly the open rows E2, E1 and the minion mods below. Resist counting (S1) is checked by `test_boss_resists` (ok in the upstream tree) and appears nowhere in the diff: equal.

## Q2. The immunities (AC #2)

`monster-mods-trace.md` rows "8 / 27 / 28 resist, spectral, stone skin" (FUN_005a1370): at most two immunities counted as the resists stand, MR +40 on fire / lightning / cold each under 100, stone skin doubles defence before the gate, enchanted +75 own element; Magic Resistant, Fire / Lightning / Cold Enchanted, Mana Burn (+20 magic), Spectral (+20 each under 75) and Stone Skin (DR +50) are run in game.exe by `umod_stats.py` over all single mods and 100,000 random sets (six resist columns): **0 differ for c / u / s**. The named uniques (Corpsefire, Bishibosh, Rakanishu, Treehead Woodfist, Blood Raven) are particular MonStats rows with fixed mod lists (uniqmods-trace.md rows 97 and the superunique table): Bishibosh (fire enchanted + MR on a level-2 shaman) is inside the single-mod and two-mod space above; Blood Raven (a superunique) inside the s rows. **Not done as named-row runs**: the two-mod uniques (E1) are the only place the equal claim fails (Q3).

## Q3. What is still open

| id | what | d2d | game.exe | tag | cases | early-game relevance |
|---|---|---|---|---|---|---|
| E2 | Cold Enchanted's chill length | the blow carries no length: `max(cold_len, 1)` = one frame chill | FUN_005a1c70 adds stat 12 x 5 + 100 (Normal level 5: 125 frames = 5 s) | V run, **fixed on the candidate: 13,283 cold_len mismatches (u + s) become 2,039** (all with two enchants on one unit, or the 49 top-level cases) | 157,618-case differential | medium: Cold Enchanted is one of the mod ids a Normal Act 1 unique can roll (I); a hit chills the player 5 s in game.exe, 1 frame on d2d (slows walk and attack) |
| E1 | two enchants on one unique (NM / Hell mostly) | one `elem` slot: the last wins | each enchant adds its own element | V run (828 elemental + 2,290 cold_len cases u) | | none in Normal (one mod) |
| M | minions carry none of the unique's mod effects (Extra Strong damage, Fast speed, enchanted damage, Mana Burn) | tagged with no mods (`objgroups.cpp`) | the mods' inits run on each party member with flag 0 | V run for the stat deltas (the harness side is a design choice, not a computed d2d value); whether the live minion list in game.exe really keeps all mods at runtime: V run of the inits, U for the live copy | 3,303 damage / to-hit + 2,928 speed + 5,706 enchanted damage + 1,867 mana burn (m single) | medium: Bishibosh / Rakanishu packs are in Act 1 |
| H1 / S1 / G1 / B | (earlier rows) | | | merged (G1, B) or no impact (H1) | | |

Residual after the E2 fix on single Cold Enchanted units: 46 (u) + 3 (s) cases, all at MonStats levels where game.exe's length uses a smaller level than d2d's final one (example row 333: game 650 = level 110, d2d 665 = level 113): I (a cap at 110 in the stat the init reads); unreachable below Hell Act 5. Listed, not claimed.

## Deviations (d2d against game.exe) and differentials

| id | tag | differential |
|---|---|---|
| D1 = E2 | V run, fix built | yes: `umod_stats.py` / `test_umod_stats` (existing, now in the fork branch) |
| D2 = E1 | V run | yes, same harness; the fix is a second element slot per boss (ai.cpp `elements` already has two) |
| D3 = M | V run (stat deltas) | partly: needs the minion tag path of `objgroups.cpp` under test; a live check is the drill below |
| D4 | the 49 top-level E2 cases | U |

## Bot findings (described, not changed)

B1 `internal/danger` / the affix model (`internal/combat/affix.go:21` `AffixResists`) skips an add that would make a third immunity; game.exe counts DR and MR too (monster-mods-trace.md B-row). The bot rule is a safe overestimate; unchanged by this card.
B2 The bot's danger model rates Cold Enchanted as low danger in Normal (monster-mods-trace.md B-rows); with E2 fixed on d2d a Cold Enchanted unique's hit chills the player for 5 s (walk -50 percent, attack slower), which is what the original game does: the rating should count the chill (not changed here).

## PR candidate

`fid/cold-enchanted-length` (fork scratch repo, private remote, pushed 2026-11-08; two commits on upstream a5b970d5): commit 1 the differential (`tools/emu/umod_stats.py`, `tests/monsters/test_umod_stats.cpp`, a README row; before: 13,283 cold_len mismatches for uniques / superuniques), commit 2 the claim (`components/rules/uniques.hpp` +6 lines `elem_len_per_level` and `boss_elem_length`, `components/game/ai.cpp` 1 line, a doc line in `docs/research/re/monsters.md`; after 2,039, of which the 1,990 with two enchants are E1 and 49 are the top-level U). Identity and message form as the TASK-83 candidate. **Not "after 0"**: the PR claim must be limited to single-enchant units below the top levels, or E1 fixed first; presubmit and pr_guard results are in the card result. Needs an independent verifier and the user's go; not posted.

Usefulness: Cold Enchanted is one of the mods a Normal Act 1 unique can roll (I); its hit chills the player; on d2d the chill is one frame. For the bot it means d2d under-rates those uniques (a soak on d2d is slightly easier than the original game there), an edge case in early play, no effect on the Sorceress's or Paladin's usual packs.

## Drill recipe (lab binary, docs/forkdbg.md)

1. E2: `debug spawn` a Cold Enchanted unique (a `spawn` with a known mod list, monster-mods-trace.md T1) next to the Sorceress at clvl 1, `debug mon set <id> dmg <big>` is not needed; let it hit once; `debug states` on the player for state 11 and its end tick: expect (level x 5 + 100) x resisted frames on game.exe; one frame on d2d before the fix.
2. M: spawn Bishibosh (Cold Plains) with his pack, read each minion's level (+3), damage and the enchant list on game.exe and on d2d: game.exe copies the mod bytes to the minions (monster-mods-trace.md T1).
