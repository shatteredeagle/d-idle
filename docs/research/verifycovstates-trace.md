# Independent verification: COVSTATES (states and auras, first pass)

> status: card verifycovstates, 2026-11-02, Sonnet verifier, offline (emulator on the worker host, no game, no d2d slot, no Go change, nothing pushed or posted). Candidate: `docs/research/covstates-trace.md` (branch agent/covstates, commit e08a012c) and its two ledger rows (states 21 and 2). Style as `verify-cursedef.md`.

Tags: **V read** (own disassembly reading with upstream's `disasm.py`, addresses given), **V run** (own harness, own seeds, counts given; game.exe's real code run natively under the emulator, d2d's real `Fight::land` built by hand against the scratch upstream build), **D** (patched 1.14d table, listed on the worker host), **S** (upstream doc or source), **I** inferred, **U** not checked. Verdict words: VERIFIED (own counts), HOLDS WITH LIMITS, NOT VERIFIED, WRONG.

## Verdict in short

| claim of the first pass | verdict |
|---|---|
| Stun (state 21): d2d `Fight::land` equals game.exe FUN_0057aae0, 0 differ | **VERIFIED** (own 25,567 + 13,029 monster cases on real MonStats rows, 0 differ in the stun end); limit: the MonStats row offsets (+0xc boss bit, +0x32 Velocity) are S on both sides of my harness too |
| Stun length cap 250, hireling cap 13, a new stun resets the end | **VERIFIED** (32,821 monster + 7,179 player + 160 hireling game-side cases equal my own model; d2d side only for monsters) |
| D2: game.exe takes a seed step for every flag-8 monster even when the boss column or Velocity 0 forbids the stun, d2d does not | **VERIFIED** (2,316 of 2,316 such cases: d2d's seed after the call is the one it started with, game.exe's is not) |
| Poison replacement: equal for whole points a frame, differs near ties | **HOLDS WITH LIMITS**: whole-point rates 0 of 10,056 differ (VERIFIED), but the deviation is larger and **runs both ways**, and a third form (blows under one point) exists; "same skill and level equal on both sides" is **WRONG** for rolled rates (D1a..D1c below) |
| D5: a poison blow refiles event 3 at frame + 1 before the strength test | **VERIFIED** (the refile in 60,000 of 60,000 cases, 14,055 of them rejected blows); the effect on the tick phase stays I |
| Q3: FUN_00639db0 sets bit `state` in the array at [stat list +0x58]; 185 ids accepted, 7 refused; a second array 6 words on | **VERIFIED** (own probe 185 of 185, 7 of 7; also the unit flag 0xc8 bit 8 from the States row byte +0x12) |
| Q6: States.txt counts (185 rows, 55 `stat`, 32 aura, 13 curable curses ...) | **VERIFIED** (D, own count of the patched table) |
| B1 (bot reads the right array), B2 (Decrepify default 56 is Terror; `amplifydam` is no name) | **VERIFIED** (B2 against the patched States.txt and `StateByName`) |
| B3 (antidote 30 s), Q7 rows other than stun and poison (chill, curses, auras) | **NOT VERIFIED** (not read here) |
| "no deviation" for stun is not arithmetic re-derivation of d2d's formula | **CONFIRMED**: the oracle is game.exe's code (gate, roll, cap, replace); the d2d side is real `Fight::land` |
| Ledger rows states 21 and 2 earned | states 21: **earned with limits**, states 2: **earned** (TESTED-DEVIATION); rewritten here with my own counts |

## What I read first (V read, before looking at the first pass's scripts)

- **FUN_0057aae0** (stun; edi = target, stack: attacker, length; `ret 8`): length <= 0 returns. A monster (`[edi] == 1`): FUN_005a0180 with mask 8 tests word +0x16 of the monster data; if set, one LCG step on the attacker's words +0x20 / +0x24 (multiplier 0x6ac690c5, `lo*m + hi`, new high = carry), `low % 100 < 90` returns (0x57ab36..0x57ab44). Then FUN_0063e9f0(0, unit) (MonStats row at `[tables+0xa78]`, stride 0x1a8, byte +0xc ANDed with the image dword 0x6ce280) nonzero returns; word +0x32 of the row (FUN_00451f80) zero returns. FUN_0063ee90 classes 0x10f, 0x152, 0x167, 0x230, 0x231 (271, 338, 359, 560, 561) nonzero and length >= 13: length 13 (0x57ab7a..0x57ab86). Any unit then: length > 250 becomes 250. State 0x15 lookup (FUN_006256b0): none: new list ending at `game +0xa8 + length`, event 0xc filed at the end, callbacks, state bit on; present: FUN_00625310 rewrites the end (always, longer or shorter) and event 0xc is filed again. A non-monster skips the flag, boss and Velocity tests (the jump to 0x57ab88).
- **FUN_0057ac50** (poison; esi = target, stack: attacker, rate, length): rate or length <= 0 returns. For a monster target the type-3 event is removed and filed at frame + 1 **before** the comparison (0x57ac87..0x57aca8). No old state-2 list: state on, new list, event 0xc, stat 0x4a set to -rate (FUN_00627150 is a one-instruction wrapper that tail-calls FUN_006270b0, which sets; read at 0x627150 / 0x6270b0). Old list: `neg(stat 0x4a) > rate` (`jg`, signed) leaves everything; otherwise end rewritten (FUN_006260b0, a two-store function), event filed, stat set. So equal rates replace.
- FUN_00625a70 / FUN_00639db0 / FUN_00639df0 as in the first pass's Q3 (own read: bounds against `[tables+0xc4]`, array pointer `[statlist+0x58]`, the second array `ceil(count/32)` words on, States row byte +0x12 bit 0 sets unit flag 0xc8 bit 8).

My reading agrees with the first pass on every point it states; I found no reading error. The poison unit question the first pass left open (rate in 256ths a frame, d2d builds `poison = rate x len >> 8`, `rules::fire` in combat.hpp, S read) I checked in d2d's source.

## Own harness (V run)

`tools/emu/vcs_game.py` (game side, gen, compare), `vcs_d2d_test.cpp` (d2d side), `vcs_run.sh` (build + both sides), `vcs_bits.py` (state-bit probe), `vcs_dis.sh` (disassembly helper). Differences from the first pass on purpose:

1. **Real data on both sides.** The game's MonStats rows are filled from the patched table for **every class** (734 rows, byte +0xc = boss bit of the real mask 0x6ce280 plus random noise in the other bits; word +0x32 = Velocity); the d2d side uses the **loaded MonType of the same row, nothing patched per case**. The first pass overrode `boss_column` and `velocity` on d2d's one fallen per case, so the loader and the row identity were never exercised.
2. **The seed is compared case by case.** d2d's `Rng` is the same LCG (rules.hpp: same multiplier, low and high words). I set the fight's Rng to the attacker's seed given to game.exe, so the 10 percent roll is a per-case differential (the first pass compared counts: 294 against 309 stuns).
3. Natively run: FUN_0057aae0, FUN_0057ac50, FUN_005a0180, FUN_0063e9f0, FUN_0063ee90, FUN_00451f80, FUN_00625310, FUN_006260b0, FUN_00625330, FUN_006252d0, FUN_00625ce0 (list end and callback writes run for real, where the first pass hooked the end writes). Hooked (not the claim): event queue (FUN_005417d0 / FUN_00540e60), FUN_00639db0 in the stun / poison runs, FUN_00554010, list lookup FUN_006256b0, allocation FUN_006251f0, attach FUN_00626e10, stat get FUN_00625d00 and set FUN_006270b0 / FUN_00627150 (**I**: that the get returns what the set stored, no shift for stat 0x4a).
4. Cases: stun: every class x the five flag sets (0, champion 0xd, unique 8, superunique 0xa, minion 0x10), lengths from a boundary list (1, 2, 12..15, 99, 100, 248..252, 300, 999, 10000) or 1..400, an earlier stun with 0, 1, 30 or 300 frames left, random attacker seeds, 20 percent player units, 160 hireling-class cases. Poison: six groups (`same_len`: one length, rates within a few 256ths; `near`: rates within 6 of each other, different lengths; `whole`; `random`; `late`: second blow after the end; `tiny`: rate x length < 256), second blow at dt = 1, len/2, len-1, len+1, len+40 or later (dt = len left out: the end event and the blow tie).

```
tools/emu/run-on-worker.sh --ttl 2400 vcs_run.sh 40000 40000 9001     # stun 40,160 cases, poison 40,000
tools/emu/run-on-worker.sh --ttl 2400 vcs_run.sh 20000 20000 77       # second seed
tools/emu/run-on-worker.sh vcs_bits.py 185
```

### Results

| run | cases | differ |
|---|---:|---:|
| my model of FUN_0057aae0 against game.exe, monster / player / hireling, seed 9001 | 32,821 / 7,179 / 160 | 0 / 0 / 0 |
| stun end, d2d vs game.exe, all monster cases, seed 9001 (7,254 more refused by `Fight::land` before the stun code: NPC and neutral classes, excluded) | 25,567 | **0** |
| the same by group: plain 4,358 stunnable + 188 boss column + 518 Velocity 0; minion 4,333 + 229 + 497; special (champion, unique, superunique) 13,128 + 651 + 1,665 | 25,567 | 0 in every group |
| stun end, seed 77 (3,701 refused) | 13,029 | **0** |
| seed after the call, special stunnable rows | 13,128 | 0 (same LCG, one draw) |
| seed after the call, special rows with boss column or Velocity 0 (D2) | 651 + 1,665 | all differ: game.exe drew, d2d did not |
| event 0xc filed at the list end | 30,700 | 0 |
| poison: my model against game.exe (end and rate), seed 9001 / 77 | 40,000 / 20,000 | 0 / 0 |
| poison winner, d2d vs game.exe, group `whole` / `late` | 6,700 / 6,716 (+ 3,356 / 3,297) | 0 / 0 |
| poison winner, `same_len` | 6,612 (+ 3,338) | **126** (+ 53): 122 (+ 53) d2d replaces where game.exe keeps, 4 the other way |
| poison winner, `near` | 6,638 (+ 3,337) | **455** (+ 230): 308 d2d keeps where game.exe replaces, 147 the other way |
| poison winner, `random` | 6,704 (+ 3,312) | 2 (+ 1) |
| poison winner, `tiny` (rate x length < 256, d2d's points floor to 0) | 6,624 (+ 3,358) | **4,834** (+ 2,442): d2d ignores the blow, game.exe poisons |
| poison tick event 3 refiled at frame + 1 | 40,000 (+ 20,000) | 0 (9,345 of them rejected blows) |
| state bits: `vcs_bits.py 185` | 185 set, 185 clear, 7 refused | 0 |

Total counted for the ledger: stun 25,567 + 13,029 = 38,596 monster cases; poison 40,000 + 20,000 = 60,000 cases.

## Deviations d2d versus game.exe (from this verification)

| # | rule, game.exe | rule, d2d | tag | differential |
|---|---|---|---|---|
| D1a | poison replacement compares the integer rate a frame in 256ths: `old > new` keeps | compares `floor(rate x len / 256) / len` as a real number, `>=` replaces: two blows of **one length** whose rates differ by less than 256/len points floor to the same points, so d2d replaces where game.exe keeps the old (122 of 6,612 same-length cases, + 53 of 3,338) | V run | yes: `vcs_run.sh`, same_len group (this card) |
| D1b | the same, lengths differing | floors in both directions: 308 keep-where-game-replaces, 147 the other way (of 6,638) | V run | yes: near group |
| D1c | a blow with rate x length < 256 poisons (rate > 0 is all that is tested) | `Blow.poison` is floored to 0 and `land` ignores a blow of 0 points | V run | yes: tiny group; real blows rarely are that small (**I**) |
| D1d | (the first pass's D1) | unchanged: whole-point rates equal (0 of 10,056) | V run | yes |
| D2 | one seed step for every flag-8 monster | none when the type is a boss column or Velocity 0 | V run | yes: seed after the call (2,316 of 2,316) |
| D3 | hireling-class monsters cap a stun at 13 | no merc stun path (`tick.cpp` only sets `actor->stunned`, the player's) | V run game side (160 cases), S d2d | no d2d side; build: strike a merc through `Fight::hurt_side` with `stun_ticks`, read `merc_*`, oracle FUN_0057aae0 with class 271 |
| D4 | player stun min(length, 250) | `actor->stunned = now + min(ticks, 250)` (tick.cpp:533, S read) | V run game side (7,179), S d2d | d2d player path not run; build as D3 |
| D5 | poison blow refiles event 3 at frame + 1 | no tick restart | V run game side (60,000), effect I | the type-3 handler is not run |

Impact of D1a..D1c is small and **not measured**: d2d's blow rate is `roll x len >> 8` floored (combat.hpp), so ties need two blows within 256/len units of each other (for len 100, about 2.6 of a roll range that is usually thousands wide), **I**. Real play can hit D1a (the same skill rolls its rate between elo and ehi), so "blows of the same skill and level are equal on both sides" in the first pass is not right.

## Claim by claim against the first pass

- **Q1 (stun code)** VERIFIED by reading and run; the first pass's table matches mine row by row. The Velocity word and boss bit offsets are **S** (upstream `skills.md`, d2d's comment), used the same way by both harnesses: an error in them would pass both. I did not read the MonStats loader.
- **Q2 (d2d stun)** VERIFIED, and stronger here (real rows, per-case seed). One check the first pass could not make: d2d's loaded boss column and Velocity per name against MonStats.txt: 733 names, **6 differ** (willowisp8, vampire9, skmage_fire7, overseer6, skmage_pois7, skeleton8): for each, d2d's value is the value of the **row above** in the table (e.g. overseer6 gets cr_lancer8's 8). That is d2d's own documented reproduction of game.exe's compiled Id lookup (comment at `gamedata_load.cpp` line 165, S upstream `bugs.md`: after MonStats' second `cr_lancer8` every later Id lands one row early); the duplicate Id row is `cr_lancer8` at rows 617 and 723 (V run). Not a deviation; a limit of any by-name comparison (it did not change a stun end).
- **Q3 (state bits)** VERIFIED. The unit-update mark is called once per accepted call.
- **Q4 / Q5 (poison)**: Q4 VERIFIED (60,000 cases, my model equals the native run; event 3 refile VERIFIED); Q5's table is HOLDS WITH LIMITS, see D1.
- **Q6** VERIFIED (D): ids 0..184 = 185 rows; `stat` filled in 55, `aura` 1 in 32, `curse` set in 26, `curable` 1 in 13 (9, 19, 23, 27, 55, 56, 57, 58, 59, 60, 61, 113, 114), `group` 13, `setfunc` 26, `remfunc` 18, `skill` 2 (redeemed 99, holyshield 101); the early rows named there (3, 4, 5, 9, 10, 11, 19, 21, 32, 34, 35, 44, 46, 48, 55, 60) have the stat or the empty cell the first pass says (prayer 34, holyfire 35, meditation 48, ironmaiden 55, stunned 21: no `stat`).
- **Q7, Q8** NOT VERIFIED (a map; the claims about chill, curses and auras were not re-run here).

## Bot findings

- **B1** VERIFIED fine: `internal/d2/states.go:20` (`slStateFlags = 0x58`) and `:26` (`Has`): game.exe indexes `[statlist+0x58]` as a pointer to the array and bit `state & 31` of word `state >> 5` (V read 0x625a96..0x625ab5, V run 185 of 185); `[unit +0x5c]` is the stat list (0x625a7e). The "UNVERIFIED live" comment can read V run for the offset and bit order.
- **B2** VERIFIED: `internal/d2/missiles.go:61` `Decrepify: 56`, but the patched States.txt has id 56 = terror, id 60 = decrepify, id 24 = slowed (D); `missiles.go:67` looks up `amplifydam`, the table row is `amplifydamage`, and `StateByName` (`internal/data/extra.go:452`) is an exact lowercase match, so Amplify resolves only by its default 9 (right by luck). Added by me: the toy fixture the first pass names at `internal/data/data_test.go:117` / `:368` (a row "slowed" with id 56 and stat "slow") is synthetic and makes no claim about the real table; it is not wrong, but it will not catch B2.
- **B3** NOT VERIFIED (`internal/combat/hard.go:70`, `antidoteDuration = 30 s`, still S).
- **B4** HOLDS: `internal/d2/player.go:88` reads poison as a bit only.

## PR candidates

**No PR candidate.** D1a..D1c are proven with a differential on real code (before: 5,417 of 39,994 differ), but no fix exists and "after 0" cannot be shown; a fix has to carry the game's integer rate (256ths a frame) in `Blow` and compare it (it also removes D1c). D2 has a differential, no outcome differs. Nothing built in the fork checkout.

## Ledger (docs/coverage/ledger.tsv)

The first pass's two states rows are replaced by mine, with my counts: `states 21 TESTED-0-DIFF 38,596`, `states 2 TESTED-DEVIATION 60,000`; `python3 tools/coverage/coverage.py --write` regenerated `rule-coverage.md`. If the covstates branch merges first, its two lines for states 2 and 21 are to be dropped in favour of these (same ids).

## Limits and dead ends

- Offsets of the MonStats row (boss bit mask 0x6ce280 & +0xc, Velocity +0x32) and that FUN_00625d00 returns what FUN_00627150 stored are I / S; the rate unit conversion of d2d (`roll x len >> 8`) is d2d's, taken from its source, and what game.exe's real rate for a given skill is was not run (the hit-side skill traces own that).
- 7,254 + 3,701 stun cases on NPC / neutral classes: `Fight::land` returns before its stun code (not targetable, converted). The game would stun them; d2d never lands a blow on them (excluded, counted).
- The player stun, the merc stun and the poison tick events were not run on the d2d side.
- A first run printed 122 stun mismatches on classes such as cain1, kashya, golems: they were the refused NPC classes above (harness artifact, now a separate counted group).
- The `tiny` group makes D1c look big (4,834 of 6,624): it is built to sit under one point; do not read it as a rate of real play.

## Drill recipe (U to confirm later, not run)

As the first pass's, plus: **same-length poison tie (D1a)**: two Poison Javelin casts of one skill level at one zombie with `debug monstate` extended to print `poison_rate` and `poison_until` (the first pass notes no verb prints them): read whether the second cast replaces when its rolled rate was lower by less than 256/length; compare with game.exe's `old > new`.

## Matrix-cell updates (proposed text)

- `docs/research/sorc-skill-matrix.md` Frozen Armor row (b): as the first pass proposes, with "25,567 + 13,029 monster cases on real MonStats rows [V run, verifycovstates-trace.md]" and the NPC-class limit.
- Same doc, states line: "state 21 run-tested against Fight::land (38,596 cases, 0 differ), state 2 (60,000 cases, D1a..D1c both directions); 183 of 185 rows untested".

## Glossary gaps

"changed array" and "live state array" (two bit arrays behind a stat list +0x58): not in `GLOSSARY.md`. Also "refused (by `Fight::land`)" for NPC-class blows.

## Shared-file lines for the merger

See `openspec/changes/verifycovstates/merge-lines.txt`.
