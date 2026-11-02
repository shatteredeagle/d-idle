# Verify SKMELEE2: independent re-check of the player's melee blow deviations

> status: current (2026-11-02, card verifyskmelee2, verifier Sonnet). Independent of docs/research/sk-melee2-trace.md: the reading of FUN_0057b420 / FUN_0057d9b0 / FUN_0057b660 and the harness were made first, the author's trace and numbers were read afterwards (section 6). Offline: no game, no slot, nothing pushed or posted. Tags: **V read** = read in the disassembly of game.exe 1.14d; **V run** = game.exe's own function run under unicorn, or d2d's real code run in a test binary; **D** = 1.14d table; **I** = inference; **U** = unverified. No game code is pasted here.

Question: do D1 (physical kept in 256ths), D2 (Blunt weapon plus stat 122 against undead), D4 (barehanded Strength term), D5 (attack rating percent floor) and D6 (the -90 floor applied once) hold between game.exe's blow functions and a build of upstream d2decomped?

## 1. Verdicts

| Dev | Verdict | Evidence (my own cases) |
|---|---|---|
| D1 | **HOLDS** (rule); the life consumption side is not re-run | V run: FUN_0057b420 returns a non-whole value in 99.5 % of blows (19,911 / 20,000 seed 11; 19,922 / 20,000 seed 3). d2d's `Blow::frac` is 0 in every blow (0 of 30,000 blow lines each seed). On weapon blows where no other deviation can act (13,461 / 13,386 cases) d2d's whole points are exactly the floor of the game's 256ths: 0 differences. Mean game minus d2d on those with SrcDam 0x80, no skill ED, at least 1 point rolled: +0.4990 (5,313 blows, seed 11), +0.4894 (5,376, seed 3). V read: FUN_0057b7d0 never shifts the record's +8 field (only store, crit doubling, subtract). That FUN_0057dbf0 takes life in 256ths is S (earlier traces), not re-run. |
| D2 | **HOLDS WITH LIMITS** | V run: FUN_0057b660 on the exhaustive grid (undead x {no weapon, non-blunt, blunt} x stat 122 {-60,-1,0,1,50} x record ED {0,35}): 60 / 60 equal to "+50 with Blunt plus stat 122, added only when the sum is over 0, only against an undead" (16 cases add a bonus). V read: FUN_0057b7d0 calls it only for a player-type attacker against a monster. d2d: no undead, Blunt or monster-type input in `Target` / `Fighter` / `player_blow` (grep of combat.hpp: 0 hits for undead, blunt, montype). Limits: the undead test (FUN_0063e990) and the Blunt test (FUN_00629bb0 with type 57) were hooked, not run; that the Blunt test follows the Equiv chain is I. D: the chain exists in ItemTypes: scep / wand / staf -> rod -> blun, club / hamm / mace -> blun. The stat 121 and 180 terms were not run. |
| D4 | **HOLDS** | V run, barehanded blows: 6,032 / 6,068 landed (seeds 11 / 3); only 1,064 / 1,148 equal in whole points (all gap is Strength). Control (game Strength forced to 0, d2d keeps its Strength, which it ignores), no flat damage: 5,521 / 5,461 of 5,521 / 5,461 equal in whole points. So the Strength-as-damage-percent term is the whole difference there. Second part, the range rule (game: stat 21 if over 0 else 1, stat 22 if over 1 else 2; d2d: 1 + flat, 2 + flat): with flat damage > 0 only 1,252 / 3,762 and 1,285 / 3,784 equal (seeds 11 / 3). Mean game minus d2d on a barehanded blow: +4.73 / +4.81 points (random Str 10..199, so the size is the generator's, not a game value). |
| D5 | **HOLDS WITH LIMITS** | V run, to-hit FUN_0057d9b0 vs `player_blow`'s hit: 20,000 cases each seed. Hit results equal in 19,769 (seed 11) / 19,774 (seed 3). All 231 / 225 differences have a game AR' (AR + MulDiv(AR, percent, 100)) at or under 0, where d2d floors it at 1 (game: ar 0 and the shortfall added to the defense, so the chance is the 5 % floor). Of my cases 2,871 / 2,851 have AR' at or under 0, so the floor matters in about 8 % of those. One more case (seed 3: AR 43, percent -25: game 33, d2d 32) differs by the rounding of a negative percent (game truncates the share, d2d truncates the sum) and flips the roll; 1 in 20,000, 0 in seed 11. Limits: reachable only with a total percent of -100 or worse, or an AR at or under 0 (no gear here does that, I); my generator draws -150..-10 to hit the corner; mastery, and the gear terms of FUN_0057d8a0 (hooked to 0), were not run. |
| D6 | **HOLDS WITH LIMITS** | V read: FUN_0057b420 floors p (ED argument + stat 25 + Str/Dex term + mastery) at -90 once, then adds the stat 17 / 18 on top without a second floor. d2d floors the gear sum in `make_fighter`, then again with the skill ED in `player_blow`. V run: weapon cases with gear p under -90 and a skill ED: 150 / 157 and 171 / 179 differ in whole points; control with the same gear p and no skill ED (the plain Attack): 350 / 350 and 367 / 367 equal. Barehanded, same pattern (control cases, seeds 11 / 3): 449 of 717 and 429 of 755 differ (the rest equal by rounding coincidence). Limit: it needs a gear percent under -90 and a skill with ED, so it never shows in the plain Attack (ED 0) and only in a debuffed corner case (I). |

## 2. What I read (V read)

- FUN_0057b420(unit, stats flag, weapon, min, max, ED, flat, SrcDam): with a weapon, min / max are the unit's stats 21 / 22 (23 / 24 for the second hand), no floor; without, 21 if over 0 else 1 and 22 if over 1 else 2. Both are shifted left by 8, plus stat 111 shifted by 8 on both; min under 1 becomes 0x100, max at or under min becomes min + 0x100. p = ED + stat 25, with a weapon + Str x StrBonus / 100 + Dex x DexBonus / 100 (both trunc toward zero) + the mastery (FUN_00645830 mode 1), **without a weapon + stat 0 (Strength)**. p is floored at -90 once. min' = min + MulDiv(min, stat 18 + p, 100), max' = max + MulDiv(max, stat 17 + p, 100); result = min' + flat + draw below (max' - min') on the unit's seed; negative becomes 0; times SrcDam / 128 when not 0x80. It is returned in 256ths.
- FUN_0057d9b0 (to-hit): AR' = AR + MulDiv(AR, mastery + stat 119 + bonus + the stat 179 matches, 100), no floor; defense below 0 moves to the AR, AR below 0 moves to the defense; chance = AR' x 100 / (AR' + DEF) (100 when the sum is 0), x 2 x alvl / (alvl + dlvl), 5..95 clamp; the roll is the attacker's seed draw % 100 compared with it. FUN_00622560 (AR) = stat 19 + (dex - 7) x 5 + the class ToHitFactor.
- FUN_0057b660: against an undead, +50 with a Blunt weapon plus stat 122, added to the record's ED when over 0.

## 3. Harness (V run)

Mine, under tools/emu with the prefix `vm2_`: `vm2_game.py` (game.exe's FUN_0057d9b0, FUN_0057b420, FUN_0057b660 under the emulator; stat getters, weapon, hand, StrBonus / DexBonus, mastery, AR and defense functions and the post-hit events hooked, seed draws native), `vm2_d2d.cpp` (d2d's REAL `rules::make_fighter` and `rules::player_blow`, compiled against the scratch d2d build; the checkout's player_blow / make_fighter lines match upstream main's), `vm2_cmp.py`, `vm2_run.sh` (chain), `vm2_dis.py` (disassembly for reading, output never committed). Two inputs are shared by construction: the AR (FUN_00622560) is computed by the formula above, and the gear model is reading A (stat 21 / 22 at the player carry the weapon's enhanced damage once, stats 17 / 18 at the player are 0), the model d2d uses; whether the real player stats look like that is U (author's D7, not checked here). So the checks compare the RULE given identical inputs.

Case design: 20,000 to-hit cases (dex 5..199, clvl 1..98, gear AR -600..1800, percent in {-150..200}, defense -300..1500, bonus {-20..100}), 20,000 random blows (70 % weapon, mastery {0,28,61}, ED {0..300}, flat, stat 111, stat 25 {-150..100}, Str / Dex 10..199, StrBonus / DexBonus pairs, skill ED {0,50,150}, SrcDam {0x40,0x80,0xC0}), 10,000 barehanded control blows (game Strength 0, half without flat), 60 undead grid lines. The seed pair is random; blows keep only seeds whose to-hit draw is under the 95 cap, d2d's damage draw is its second and the game gets the seed one step on.

Reproduce (the worker host):

    tools/emu/run-on-worker.sh --ttl 1500 vm2_run.sh 20000 11 > /tmp/vm2-run11.txt 2>&1
    tools/emu/run-on-worker.sh --ttl 1500 vm2_run.sh 20000 3  > /tmp/vm2-run3.txt 2>&1

Runs: seeds 11 and 3, 20,000 each; 12 s a run after the first build. The ItemTypes chain: `tools/emu/run-on-worker.sh mpq-table.sh itemtypes`.

## 4. Differences from the author's numbers

- D1: mine +0.499 / +0.489 a landed blow on the clean subset (author +0.498 / +0.499); the unrestricted mean (+0.27 / +0.25) is lower because it includes SrcDam 0x40 blows whose game value is under 1 point while d2d floors them up to 1 (I, not split out); the rule agrees.
- D4: author 12,325 / 12,150 barehanded whole-point differences; mine 4,968 / 4,920 of 6,032 / 6,068 (82 %), plus the control that isolates the Strength term (the author did not split Strength from the range rule).
- D5: author 44 / 36 of 100,240; mine 231 / 225 of 20,000 because my generator puts a third of its cases at AR' under 0 on purpose; the cause (AR' at or under 0) is the same, plus one rounding case the author's list did not name.
- D6: author 38 / 40 of 380 / 376; mine 150 / 157 and 171 / 179, same cause.
- D2: same 60-line grid, same result; I add that the Equiv chain is data (D) but the walk is still I.

## 5. Limits and not covered

D3 (FUN_0057d8a0 gear terms), D7 / D8 (stats 17 / 18 at the player), crit / deadly strike (no stats set), the second hand, FUN_0057dbf0 / FUN_0057c1e0 (life in 256ths, resistances) and the mastery function itself were not run. The bot (internal/combat, internal/resource) was not re-read: for bot findings see sk-melee2-trace.md; I add none. No d2d PR candidate from this card (the verifier's job is the check, and the authors' PR decision stands).

## 6. Drill recipe (U to settle later, not run)

On a lab instance with a throwaway character: (a) barehanded blow vs a training dummy-like monster with Strength changed by stat points, read the monster's life before and after to confirm the Strength term and the 1-2 range (D4); (b) Act 1 zombie with a staff vs a sword of equal damage (D2); (c) read stat 21 / 22 / 17 / 18 on the player with the documented read verbs to settle reading A / B (D7).

## 7. Shared-file lines for the merger

See openspec/changes/verifyskmelee2/merge-lines.txt.
