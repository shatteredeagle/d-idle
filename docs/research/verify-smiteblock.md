# Verify SMITEBLOCK: does a block stop Smite in game.exe 1.14d?

> status: independent verifier (card verifysmite, Opus) of `docs/research/smiteblock-trace.md`; docs only, nothing pushed or posted.

Own harness: `docs/research/verify-smiteblock/` (vs_dis.py, vs_scan.py, vs_tab.py, vs_oracle.py, vs_sum.py, vs_env.sh; raw table `results.txt`). Run on the worker host: `labrun <worker> -- bash docs/research/verify-smiteblock/vs_env.sh vs_oracle.py 20000 7331 [unforced|dmg]`, niced, one at a time. game.exe was read and run BEFORE the author's trace was opened; seeds 7331 / 99 / 20261102 (the author used 7 and 1).

## Verdicts

| # | Claim | Verdict | Tag |
|---|---|---|---|
| 1 | FUN_005ce9f0 is srvdofunc 150 (Smite) | VERIFIED: table word at 0x732508 = 0x5ce9f0 (base 0x7322b0 + 150 x 4); skills.txt Smite Id 97 srvdofunc 150 | V run |
| 2 | calls FUN_0057ec10 at 0x5ceba8, then `or ax, 1` at 0x5cebb0 | VERIFIED (own disassembly) | V read |
| 3 | block bit 0x10 stays set, nothing on the damage path tests it | VERIFIED: FUN_0057dbf0 tests `0x8380` (0x57dc1d) and bit 0 (0x57dc24); FUN_0057d4f0 tests only bit 0 (0x57d591) before FUN_0057c6c0 | V read + V run |
| 4 | Smite always lands on a monster defender, block or not | VERIFIED: 20,000 of 20,000 end-to-end cases reach FUN_0057c6c0 with damage, incl. 4,292 blocked (0x0011) | V run |
| 5 | a plain attack with a block bit is not applied | VERIFIED as the counterfactual: the same chain with the 4 bytes at 0x5cebb0 NOPed lands 1,552 of 20,000, exactly the "to-hit passed and defense result 0" cells; 0 of the blocked | V run |
| 6 | d2d deviates (D1) | VERIFIED by reading current upstream main 82ea6db4: `components/rules/combat.hpp:513` rolls the block for Smite and returns; `components/game/fight/strike.cpp:456` ends the blow on `blocked`. The author's line numbers (499 / 303, 412) are STALE | V read; d2d build NOT RUN |
| 7 | contradicts palskill D6, agrees with PR 23's D8 | HOLDS (on the effect) | S |

Overall: **HOLDS WITH LIMITS** (game side VERIFIED, d2d side read not run). As a candidate: **NEEDS CHANGES** (line refs, the animation / get-hit half of the fix, a d2d run).

## 1. game.exe, own run (V run)

vs_oracle.py runs FUN_005ce9f0 itself (the author built the record by script and did not run it end to end), and natively inside it FUN_0057b420 (damage), FUN_0057ec10, FUN_0057dbf0, FUN_0057ca80, FUN_0057d4f0. Hooked per case: to-hit FUN_0057d9b0 (hit / miss), defense FUN_0057dfb0 (0, 1 block, 2 / 4 / 0x10, 3), the resist step FUN_0057c1e0 (counted, leaves a marker), the consumer FUN_0057c6c0 (prints the record), stats, the shield record bytes +0xfe / +0xff, the skill calcs. Defenders: monster (type 1), player normal, player in mode 3.

Counts (results.txt): forced 20,000 / 20,000 land; unforced 1,552. Record word at the consumer: miss -> 0x0001 (no defense roll), hit -> 0x0005, block -> 0x0011, avoid bits -> 0x0101 / 0x0081 / 0x8001. Bit 4 (0x0004) is only set on a clean hit: a blocked or missed Smite arrives without it (U what bit 4 drives; see fix). Player defender in mode 3: no to-hit roll (auto-hit at 0x57ec77).

**O1 answered (V run + V read):** with an avoid bit (0x100 / 0x80 / 0x8000) FUN_0057dbf0 skips FUN_0057c1e0, FUN_0057d4f0 calls FUN_0057c6c0 with its resist argument 0 (0x57d5b9), so the RAW damage applies: the record's rec+8 equals FUN_0057b420's return in all 20,000 cases, resist step 0 times in every avoid cell. FUN_0057c6c0's life path (0x57c7f8 clamp to stat 6, FUN_0057c420, stun FUN_0057aae0 with rec+0x44 at 0x57c89b) has no hit-bit gate; only the on-hit events (0x57c7ad) test bit 0. Reach: avoid bits come from FUN_0057dfb0's player branches (PvP / a player defender), not a monster (author's reading; I did not run FUN_0057dfb0).

**Damage formula (V run, own probe of FUN_0057b420 with Smite's arguments, 400 rolls per row):** damage (256ths) = roll in [shield_min<<8 (+ Holy Shield min), shield_max<<8 (+ Holy Shield max)] x (100 + calc1 ED + str x StrBonus / 100) / 100; all 20,000 chain cases inside the bound (0 out). Quirk: with min == max the roll still spans +255 (10-10 -> 2560..2814; 1-1 -> 256..511), while d2d's `high > low ? ... : low` returns exactly low: a shield with equal min / max at >100 % can deal 1 more in game.exe (10-10 at +100 %: game 20..21, d2d 20). Low, separate from D1. Not run: dex bonus, off-weapon ED (d2d adds `kDamagePercent` and dex in `smite_pct`, combat.hpp:135).

Undead / boss: the Smite chain never queried the undead test FUN_0063e990 (0 of 20,000) and reads no monster class data besides type and the alive / attackable checks, so bosses and undead get the same rule (V run for the queried set; FUN_0057dfb0 interior, where the block chance lives, hooked: block 0 / partial / 100 % are the outcome cells, not the chance formula: NOT RUN).

## 2. Which other skills force the hit bit (V run, whole-binary scan)

All 33 call sites of FUN_0057ec10 (vs_scan.py): **only 0x5cebb0 (Smite) ORs bit 0**. Zeal (do 13, FUN_005dbc60): `test al, 1` -> block stops it. Sacrifice (do 64, FUN_005ce8e0): no FUN_0057ec10 call, takes the queued record of its start (FUN_0057d690) -> block stops it. Charge (do 67, FUN_005cf900): player attacker `or ax, 8` (knockback only) -> block stops it; a non-player charger gets a fixed word 9 (0x5cfc30, forced hit). Blessed Hammer (do 73, FUN_005d0040): no FUN_0057ec10 call (missile path; out of scope). So a Smite-only fix in d2d is right; no other Paladin melee skill should change.

## 3. Proposed d2d fix (not pushed; NOT built or run here)

```diff
--- components/rules/combat.hpp (player_blow, line 513)
-    if (target.block > 0 && rng(100) < target.block) { blow.blocked = true; return blow; }
+    if (target.block > 0 && rng(100) < target.block) {
+        blow.blocked = true;           // Smite: FUN_005ce9f0 ORs the hit bit back (0x5cebb0); the block only turns the animation
+        if (!swing.smite) return blow;
+    }
--- components/game/fight/strike.cpp (Fight::land, line 456)
-        if (blow.blocked) { block_anim(*game_data, target, now_ms); return false; }
+        if (blow.blocked && !blow.hit) { block_anim(*game_data, target, now_ms); return false; }
```
Keeps the block roll (same rng draw count for every skill, so other d2d tests keep their streams). Still needed (U, not read far enough): after the damage, a blocked Smite without knockback should play the block animation instead of a get-hit (FUN_0057cee0 tests knockback first, then 0x10 -> mode 0x13; and the record lacks bit 4). Tests: Smite vs `Target{.block = 100}` -> hit, blocked, damage > 0, stun and knockback kept; plain Attack and the Zeal / Sacrifice swings vs block 100 -> not hit (unchanged); block 0 -> byte-identical blows before / after; count of rng draws unchanged. Other `player_blow` callers (pets.cpp, missiles.cpp, deaths.cpp) never set `swing.smite`, so `blocked` keeps meaning "not hit" there.

CI parity (by reading the diff): no new local name (no C4456 shadowing), no `min` / `max` / windows.h macro names, no new include, no struct change (`Blow::blocked` exists). Low risk.

## 4. Bundle interaction

D1 melee 256ths: the Smite branch already works in 256ths and returns before the crit / SrcDam code, untouched by the diff. D2 undead bonus: game.exe's Smite path never calls the undead test (above), so the bundle's undead term must not reach Smite (check that it stays in the non-smite branch). Freeze rules: Smite carries no cold; whether a frozen monster can block sits in FUN_0057dfb0 / d2d's block gate, unchanged by the diff (NOT RUN). The fix touches two lines disjoint from those components; it can go in the next batch after a d2d run.

## 5. Against the author (read after the run)

Agree on every rule and address, on O1's direction and on Zeal-type skills. Differences: the author did not run FUN_005ce9f0 end to end nor the damage formula (done here); the author's d2d line numbers are stale against upstream main 82ea6db4; the author did not scan the other FUN_0057ec10 callers (done: Smite is the only forcing site). Not done here: the d2d build and probe (base vs expected counts), FUN_0057dfb0 / FUN_00622720 interiors, the animation / get-hit after a blocked Smite.
