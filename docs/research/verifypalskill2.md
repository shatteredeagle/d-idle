# Verify PALSKILL: independent re-check of the Smite, Zeal and Sacrifice trace

> status: current (2026-11-02, card verifypalskill2, verifier Sonnet). Offline: no game, no slot, nothing pushed or posted. Order of work: I read FUN_005ce9f0 (Smite do) and FUN_005ce8e0 (Sacrifice do) in the disassembly and the three skill rows of the patched table first, wrote my own generator and model, ran game.exe's code, and only then compared with docs/research/palskill-trace.md. Tags: **V read** disassembly of game.exe 1.14d; **V run** game.exe's own function under the emulator, compared with d2d's real rules (the existing probe `pal2_test`, built from upstream sources); **D** patched table; **I** inference; **U** unverified. No game code is pasted.

## 1. Verdicts

| Claim (trace) | Verdict | Evidence |
|---|---|---|
| Calc slots 1..4 and the to-hit bonus of Sacrifice, Smite, Zeal equal d2d (40,720 cases) | **VERIFIED** (calc part, my own count) | V run: FUN_00646ca0 for the four slots and FUN_006449f0, 8,000 random cases per seed (skill, level 1..80, 0..6 synergy levels, clvl), seeds 5 and 23: 0 differ from d2d's `eval_calc` / `skill_tohit`, and 0 differ from my model written from the table text (D: Sacrifice 180 + 15 per level + 15 x Redemption + 5 x Fanaticism, price 8; Smite 15 per level, stun min(250, 15 + 5 per level); Zeal min(2 + lvl - 1, 5) hits, ED (lvl - 4) x 6 above level 4 + 12 x Sacrifice; AR bonus ToHit + LevToHit x (lvl - 1)). Limit: d2d's side sets stat getters to 0, game side clvl only; both ignore other stats (shared, as in the trace). |
| D1 Smite carries stat 111 in game.exe, d2d ignores it | **VERIFIED** | V run: FUN_0057b420 with edx = 0 and the shield as item, 8,000 cases per seed with no negative percentage. With stat 111 > 0: 3,968 of 3,992 (seed 5) and 3,998 of 4,018 (seed 23) differ from d2d's `player_blow` in whole points; with stat 111 = 0 and min < max: 0 of 3,589 and 0 of 3,593. The game value lies inside my bounds model (min and max both + stat 111 x 256, min floored at 1 point, then the percent) in 8,000 of 8,000 both seeds. The oracle is game.exe code (the damage function is called natively; the draw is the unit's seed), not d2d's formula re-derived. Limit (shared with the author): stat 111 is fed as an arbitrary player stat; that an equipped item's 111 sits in the player's list when Smite runs is U. Reach (three uniques, one runeword) not re-counted (D, author's number). |
| D2 max forced to min + 1 point | **VERIFIED** | V run: the min = max cases (419 of 8,000 seed 5 with stat 111 = 0; 389 seed 23) differ in 274 and 257 whole points (the rest equal by rounding); the game value is inside my bounds model with max = min + 1 point: 0 outside. Reach: Zakarum Shield only (D, author). |
| D3 / D4 / D5 (floor -90, rounding, 256ths) | **NOT RE-RUN** here (my generator has no negative percent). D5 is seen: exact-256ths equality is 10 and 13 of 8,000, as the trace says; D3/D4 are SKMELEE2 results already re-checked in docs/research/verify-skmelee2.md (D6 there). |
| D6 Smite forces the hit (`or ax, 1` at 0x5cebb0); d2d has no roll; not a deviation | **HOLDS (V read only)** | I read 0x5ceba8..0x5cebb4: the call to FUN_0057ec10 with two zero pushes is followed at once by `or ax, 1` on the return value, then the join at 0x5cebe9 tests bit 0 and ORs the skill's ResultFlags (+0x12e) and +0x130 into the record. Not run (the whole function needs the party and record machinery). The roll draws one random number: the stream only. |
| D7 no shield: game.exe returns 0 | **HOLDS WITH LIMITS (V read)** | 0x5cea6e..0x5cea96: for a type 0 unit, FUN_0063c8f0 (inventory pointer at unit +0x60) fails -> jump to the return of 0; item pointer null -> argument -1 to FUN_006335f0, 0 -> the same return. Limits: that FUN_0063c8f0 returns the shield and not "any hand item" is I (the author says I too); whether the client blocks the select first is U. |
| D8 Sacrifice price reads the life when the do runs | **HOLDS WITH LIMITS (V read)** | FUN_005ce8e0: target FUN_00553540; record FUN_0057d690 (caster, target) -> dealt = record +8; the life is read now with FUN_00625480 (stat 6) and the smaller of the two is used; the price is `MulDiv(that, calc2, 100)` (FUN_00483360 takes the 100 as its one stack argument: the push 0x64 is its, not FUN_00646ca0's, which returns with `ret 0x10`, four arguments; I misread this at first); applied to the caster with record flag 0x1000. Whether the target's life is already post-hit when the do runs is U (the start function applies the hit through FUN_0057dbf0; the order of the two calls is not read here). |
| D9 Zeal reach | **NOT VERIFIED** | not read. |
| Q1 to Q6, rows of the table | **HOLDS** for what I re-read: Sacrifice srvstfunc 29 / srvdofunc 64, no mana, ToHit 20 + LevToHit 7; Smite srvdofunc 150, mana 2, ResultFlags 8, itypea1 shld; Zeal srvstfunc 37 / srvdofunc 13, mana 2, ToHit 10 + LevToHit 10, reqskill Sacrifice (D). Smite's call into FUN_0057b420 (0x5ceb2d..0x5ceb31): edx = 0, item = the shield, min / max the row bytes +0xfe / +0xff << 8 plus Holy Shield's two damages when state 0x65 is on, ED = calc1, flat 0, SrcDam 0x80; stun calc2 into record +0x44 (0x5ceb4b..0x5ceb52 stores `[ebp-0x44]`); the block bits are tested by FUN_0057dbf0. All as the trace says. |
| Bot findings B1 to B5 | **HOLDS WITH LIMITS** | Cited lines re-found: `profiles.json` bands (Zeal, Smite, Attack; Smite, Attack), `rules.go:237` (`data.Physical: Zeal, Smite, Sacrifice`), `fighter.go:717-720` (hold only for skills flagged hold), `controller.go:375` HoldClick; no shield check in `internal/combat` (grep for "shield": only comments and the Barbarian pair note). `bindleft.go:15` is a comment about the popup geometry, not a skill list: the "plain left skill" statement rests on the geometry comment only. |

## 2. A wrong turn of mine (kept)

My first Smite model added Strength itself (the barehanded rule of FUN_0057b420). It put the game value outside the bounds in 4,463 of 8,000 cases. Reading again: Smite passes the shield as the item, so the Strength / Dex term is the item's StrBonus / DexBonus percent of them, 0 for a shield row. With that the model holds on 16,000 of 16,000. So the trace's d2d side (shield row with `str_bonus` 0) is right and Strength does not scale Smite on game.exe (V run, shield StrBonus 0 only; a shield with a bonus was not run, I).

## 3. Harness (V run)

Mine: `tools/emu/vp2.py` (generator, game side, model, comparison) and `tools/emu/vp2_run.sh` (chain). The game side needs the same emulator oracle and the same stat-getter hooks as the author's (`skill_damage.Oracle`, hooks of the stat and item getters: shared inputs, not the claim); the d2d side is the author's probe `pal2_test` (d2d's real `eval_calc`, `skill_tohit`, `make_fighter`, `player_blow`), so d2d's code is not re-derived by me. My model is written from the table text and my reading, not from d2d. Seeds 5 and 23, 8,000 C and 8,000 S cases each (16,000 lines per case file; md5 first 8: seed 5 cases 6052090d, game side f4a37f9e).

```sh
tools/emu/run-on-worker.sh --ttl 3000 vp2_run.sh 8000 5  > /tmp/d2agent-pa-verifypalskill2/out5.txt  2>&1
tools/emu/run-on-worker.sh --ttl 3000 vp2_run.sh 8000 23 > /tmp/d2agent-pa-verifypalskill2/out23.txt 2>&1
```

## 4. Differences from the author's numbers

The author's counts are over a different generator (negative percentages, Holy Shield, random Strength through the shield's bonus pair): D1 4,475 of 4,500 / 4,466 of 4,487 (mine 3,968 of 3,992 / 3,998 of 4,018), D2 367 of 500 (mine 274 of 419 / 257 of 389). Same pattern, same cause. The C count matches in kind (0 differ). Nothing in the trace was wrong in what I could re-run.

## 5. Limits, not covered

D3, D4, D5 roots, Holy Shield's damage numbers, Zeal's and Sacrifice's weapon damage (FUN_0057b7d0), Zeal's reach, whole functions FUN_005ce9f0 / FUN_005ce8e0 / FUN_005dbc60 (no whole-function run: only the cores). "No deviation" claims that the trace makes (calc slots, to-hit bonus) are game.exe code against d2d code, not d2d's formula alone. D6's "not a deviation" rests on a read of one instruction. PR candidate: none from this card; the D1 before / after belongs to the trace and now has an independent before count (the after run with the candidate change was not repeated here).

## 6. Drill recipe (later, lab; not run)

Same as the trace's R1 to R4 (docs/forkdbg.md). For D1: skip until an item with stat 111 exists. For D8, R1 shows only d2d's reading; for game.exe the emulator recipe E1 of the trace stands (hook the stat getter, call the start function then the do).

## 7. Shared-file lines for the merger

`openspec/changes/verifypalskill2/merge-lines.txt`.
