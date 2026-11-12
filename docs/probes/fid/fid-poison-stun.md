# fid-poison-stun (decomp batch 7, matrix rows SK08 / MA11 neighbours): poison stacking and stun end, d2d against game.exe, re-run

> status: finished 2026-11-11 by agent decomp11 (Sonnet), offline (emulator on the worker host; no game, no live d2d, nothing pushed or posted, no fork branch). Result: **re-run of the existing differential (`tools/emu/vcs_run.sh`, seed 5151, 20,160 stun and 20,000 poison cases): the stun END is equal (13,066 monster cases, 0 differ), the stun's seed draw still differs for boss-column and Velocity-0 rows (1,165 cases), and the poison REPLACEMENT rule still differs from game.exe in 2,629 of 19,997 sequences (the 2026-11-02 run said 5,417 of 39,994: same shape, same groups).** Nothing new in the rules, so no new PR candidate; this card re-confirms the numbers on a fresh seed and adds the per-frame question (below). Follows [verifycovstates-trace.md](../../research/verifycovstates-trace.md) and [covstates2-trace.md](../../research/covstates2-trace.md); status page [fid-status-effects.md](fid-status-effects.md) listed poison as "not re-run".

Tags: V run, V read, D, S, I, U as in batch 5. Scripts: `tools/emu/vcs_run.sh` (a card of earlier work, run unchanged: `tools/emu/run-on-worker.sh --ttl 2400 vcs_run.sh 20000 20000 5151`). The d2d side is d2d's real `Fight::land` built by hand against the scratch checkout `sorcrest-d2d` of the worker host (head not recorded here, U); the numbers equal the 2026-11-02 ones group by group, so no d2d commit since then changed these two rules.

## Q1. Stun: length, cap, replace, the 10 percent shrug-off (V run, equal)

game.exe FUN_0057aae0 (verifycovstates Q1): length <= 0 does nothing; a monster whose type word has the "special" flag (+0x16 mask 8: champion / unique / superunique) takes one LCG step on the attacker's seed and shrugs the stun off when `low % 100 < 90`; boss-column and Velocity-0 monsters are not stunned; the stun end is `now + min(length, 250)` for monsters (hireling class 13), a new stun resets the end, state 21, event 0xc filed at the end.

Re-run (seed 5151, per-case attacker seed given to both sides):

| claim | cases | differ |
|---|---|---|
| model vs game.exe, monster / player / hireling | 16,726 / 3,274 / 160 | 0 / 0 / 0 |
| stun end, d2d vs game.exe, all monster cases | 13,066 (3,660 more refused by `Fight::land` before the stun code: NPC / neutral classes, excluded) | **0** |
| by group: plain / minion / special, stunnable row, boss column, Velocity 0 | 2,288 + 104 + 279 / 2,221 + 110 + 245 / 6,654 + 331 + 834 | 0 in every group |
| event 0xc filed at the end | 14,624 | 0 |
| classes with an end mismatch | 0 of 572 | |
| seed after the call, special + stunnable | 6,654 | 0 |
| seed after the call, special + boss column / Velocity 0 | 331 + 834 | **all differ** (D2: game.exe draws, d2d does not) |

Stun "recovery" (what the monster does after the stun) is the freeze-resume path shared with the frozen monster (fid-status-effects Q3, V run, equal) and the get-hit path (fid-hit-recovery.md, equal): the stun state ends, the monster thinks again in its walk / run mode at once (FUN_005a8030, WL and RN only). Not run for stun itself: the hit-recovery animation of a stunned monster (U).

## Q2. Poison: stacking rule, length, damage (V run, DEVIATES)

game.exe FUN_0057ac50 (verifycovstates Q4/Q5, V read + V run): rate or length <= 0 returns; **for a monster target the poison tick event (type 3) is removed and re-filed at frame + 1 before the strength test**; no old poison: state 2 on, stat 0x4a (hpregen) := -rate (rate in 256ths a frame), length added to the frame counter, event 0xc; an old poison: **`neg(stat 0x4a) > rate` keeps the old, otherwise replace** (equal rates replace, rates compared as whole 256ths a frame). d2d builds `Blow.poison = rate x len >> 8` (whole points over the whole length) and compares the real number `floor(rate x len / 256) / len` with `>=` (the combat.hpp rule), then keeps one `Regen` entry.

Re-run (seed 5151, 20,000 sequences of two blows):

| group | cases | differ |
|---|---|---|
| model (my copy of the game rule) vs game.exe, end and rate | 20,000 | 0 |
| winner, d2d vs game.exe, ALL | 19,997 | **2,629** |
| `whole` (rates in whole points) / `late` (second blow after the end) | 3,487 / 3,330 | 0 / 0 |
| `random` | 3,272 | 1 |
| `same_len` (one length, rates within a few 256ths) | 3,376 | 67: d2d replaces where game.exe keeps (66), the other way (1) |
| `near` (rates within 6 of each other, different lengths) | 3,265 | 211: d2d keeps where game.exe replaces (139), the other way (72) |
| `tiny` (rate x length < 256: d2d's points floor to 0) | 3,267 | **2,350**: d2d ignores the blow, game.exe poisons |
| event 3 re-filed at frame + 1 (rejected blows: no / yes) | 15,322 / 4,678 | 0 / 0 |

The deviations are the earlier D1a (same-length tie), D1b (near rates, both directions), D1c (blow under one point) and D5 (the re-file; V run on the game side only). Real size: whole-point rates, the common case, are equal (0 of 6,817 in `whole` + `late`); `tiny` needs rate x length below one point, which real skill poison (Poison Dagger, Poison Javelin, Plague Javelin, monster poison missiles, rates in 256ths of a point a frame over 100+ frames) rarely produces (I; not measured); ties (`same_len`) need two blows of the same skill inside 256/len of rate, 2 percent of the same-length group here (V run), more often in a real fight because the same skill rolls its rate between its min and max (I).

### Per-frame damage and the poison tick (U, shared with batch 6)

What the stat 0x4a does each frame is the unit's regen event (type 3), the same handler that applies healing potions (`hpregen` in their own state; re-filed by FUN_0057ac50 for a poisoned monster). Upstream's note (re/combat.md "Poison and burning") describes it as minus the per-tick damage, one tick a frame, ending through FUN_0056e900. **The event handler itself was never run**: the poison part of the differential stops at the state (rate, end, which blow wins). d2d applies one `Regen { -poison/40 per ms, until }` entry per actor (tick.cpp:524-528, scratch tree) and, for a merc, `poison / 256 / 40` per ms (tick.cpp:487). Because the same handler carries potions, the open ST05 card of batch 6 (potion effect, chain drinking) will have to run it; **proposed: that card also runs a poisoned unit for N frames and prints life per frame** (oracle: life after each frame; inputs: rate, length, a potion in the same window). Until then the claims "poison does rate/256 points a frame for length frames" and "d2d's total = rate x len >> 8" are V read (statements of the state) and I (the frame loop), not V run.

## Deviations (d2d against game.exe)

| id | what | game.exe | d2d | tag | differential |
|---|---|---|---|---|---|
| D1 | poison tie at the same length (D1a) | integer rate in 256ths, `old > new` keeps | real number `floor(rate x len / 256) / len`, `>=` replaces | V run (this card: 67 of 3,376) | yes, `vcs_run.sh` |
| D2 | poison near rates, different lengths (D1b) | as D1, whole 256ths | as D1 | V run (211 of 3,265, both directions) | yes |
| D3 | blow under one point (D1c) | poisons when rate > 0 | `Blow.poison` floors to 0, `land` ignores it | V run (2,350 of 3,267) | yes |
| D4 | event 3 refile (D5) | re-filed at frame + 1 before the strength test, also on a rejected blow | no tick restart | V run game side (20,000), effect on the tick phase I | the type-3 handler is not run |
| D5 | stun seed draw for boss-column / Velocity-0 specials | one LCG step then no stun | no draw | V run (1,165 of 1,165) | yes (seed after the call); the effect is a changed random stream for the attacker, not a different outcome of that blow (I) |
| D6 | merc / player stun | hireling class cap 13; player min(len, 250) | merc: no stun path; player: min(ticks, 250) (tick.cpp) | V run game side (earlier), S d2d | game side only |

## Bot findings (described, not changed)

B1 No bot decision reads a monster's stun or poison: `internal/combat` and `internal/danger` have no reference (grep of the same terms as fid-status-effects B1). A "kite a stunned monster" or "do not stack poison" rule would rest on the equal rules (stun end, length) and not on the deviating ones (poison replacement), so it is safe today. Consumer: none needed.
B2 Early sources are the Paladin's Smite stun (record +0x44) and the Necromancer's poison skills; any future poison model must use the rolled rate (it varies between the skill's min and max), not the average, for the tie risk (I). Consumer: the Necromancer card family (no card yet).

## PR candidate

None. D1 to D3 have a differential with mismatches before (2,629 of 19,997), but no fix exists on a fork branch and "after 0" cannot be shown; a fix has to carry the game's integer rate (256ths a frame) in `Blow` and compare it (it also removes D3), and the caller sites (`fight.cpp`, tick.cpp merc path) change with it. Not built here (Opus-verifier scope).

## Drill recipe (lab binary, docs/forkdbg.md; writing only)

P1 (D1, D3): `debug spawn zombie 1`, give it life 5000, two Poison Javelin casts of the same level in a row; read the monster's poison fields with `debug monstate` (needs the poison_rate / poison_until print listed in covstates2 as missing). Game.exe expects the second cast to replace only when its rolled rate is not lower. P2 (per frame): the same monster, one cast, `step 1` repeatedly and read the life: expect a constant drop of rate/256 points a frame carried as a fraction, with the whole-point crossing on some frames. S1 (stun): `debug spawn` a champion and hit with a stunning skill N times; count stuns of N (expect 10 percent of landed stuns, a random seed rule, D5 does not change the count).

## Matrix-cell update (proposed text)

SK08 / MA11: "UPDATE 2026-11-11 (decomp11, fid-poison-stun.md): vcs_run.sh re-run, seed 5151: stun end 13,066 monster cases 0 differ (V run); poison winner 2,629 of 19,997 differ (same_len 67, near 211, tiny 2,350, random 1; whole and late 0), D1 to D3 unchanged. Per-frame regen handler (event 3) still not run (shared with ST05)."
