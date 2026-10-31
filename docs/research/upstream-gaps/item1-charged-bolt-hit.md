# Item 1: Charged Bolt hit and damage path (agent UPSTREAMGAPS, 2026-10-30)

Tags: V = run or read by this agent (command given), S = upstream doc, U = unverified. Target: classic 1.14d. No game data or decompiled code here: addresses, function names, numbers only. Nothing was pushed or posted.

## Verdict: d2d MATCHES on ranges and bolt counts, DIFFERS on fractional damage

### What matches (V)
- Per-bolt range and bolt count from game.exe's own tooltip code (upstream `tools/emu/skill_tips.py 20 0`, FUN_004edea0) equal d2d's (`test_game --skill-tips 20 0`): zero diff over all 22,800 lines; for skill 38 (Charged Bolt): bolts 3/6/10 at level 1/4/8 (and 22 at 20), Lightning Damage 2-4 / 3-5 / 5-7 at level 1/4/8 (13-15 at 20). Saved: `cb-tooltip-gameexe.txt`. Unit-less run: synergy not included.
- Skills.txt row 38 (V, patched stack): srvdofunc 17, srvmissilea chargedbolt, calc1 `min(24,ln12)`, Param8 6, HitShift 7, EType ltng, EMin 4 / EMax 8, EMinLev1..5 1/1/2/3/4, EDmgSymPerCalc `(skill('Lightning'.blvl))*par8` = 6 % per Lightning level. Missiles.txt row 56 chargedbolt: no ToHit, no pSrvHitFunc, no pSrvDmgFunc, CollideType 3, CollideKill 1, Size 1, Range 98. So the hit needs no to-hit roll and the bolt ends on its first unit (d2d: CollideKill 1 path).
- The elemental roll (V, oracle): game.exe FUN_005a89a0 (a missile's damage record) rolls lightning as `min + rand(max - min)` on the MISSILE's own seed (unit +0x20, FUN_0045c3e0), via FUN_005a8910: stats 0x32/0x33 (min/max, 256ths), a swap when min > max, the lightning-mastery stat 0x14a added to both ends by MulDiv(x, pct, 100), zero when either stat is 0 or less. Record field +0x1c. `missile_roll.py` (this folder; goes in upstream `tools/emu/`) runs the real function on 3,000 random cases and equals the port: output `missile_roll.out.txt` (`ok`). d2d's `Rng::range(lo, hi)` is inclusive (`lo + rng(hi - lo + 1)`), game.exe's top is exclusive: 1/256 hp difference, not material.
- Hit chain read from the disassembly (V, `disasm.py`): FUN_005adf10 (unit hit) skips a NextHit already struck (state 0x56), runs the to-hit test FUN_0057d9b0 only if the row's ToHit (+0x18b) is nonzero, the hit function (table 0x73c840) from +0xe, builds the record (FUN_005a89a0), the damage function (0x73c960) from +0x10, then FUN_005adcd0 -> FUN_005ad730 (sets record flag 1, the avoid/block test FUN_0057dfb0, then FUN_0057c6c0 with the record). CollideKill (+0x186) sets end flag 1.

### What differs (V read + S): damage is cut to whole hit points
- game.exe: the damage record's total is in 256ths (record int 0x13, read at 0057c852) and is taken off the life stat 6, also in 256ths; only "life < 0x100 after the hit" is dead (0057c865, set to 0 at 0057c874). S: upstream `docs/research/re/combat.md` "Applying damage" says the same; `monsters.md` says stats 6/7 are in 256ths.
- d2d: `rules::missile_blow` does `rng.range(elo, ehi) >> 8` then `resisted(..)` in whole points; `Monster::hit_points` is an int of whole points, `hurt()` subtracts a whole int. A Charged Bolt of 2-4 (512..1024 in 256ths) therefore deals 2 or 3 (mean 2.5), where game.exe deals mean 3.0 (about 17 % less per bolt at skill level 1; about 8 % at level 8, 5-7 (mean 6.0 vs 6.5 after the 0.5 rounding of the displayed range); the loss is the same 0.5 hp per hit before resistance for every spell and every other element). Also a hit that rolls under 1 hp does nothing in d2d (`hurt` returns at damage <= 0, wakes nothing).
- Why it matters to us: Sorceress clvl 11 to 14 kills Act 1 monsters (Fallen 6 to 30 hp) with 5 to 14 hit-point bolts; shaving 0.5 of each is a few percent of kills per hour (I: not measured; the exact effect needs the live comparison).

### Not covered (U)
- Synergy on the real unit path (EDmgSymPerCalc through the skill-level lookup): no unit-level oracle run; d2d has a unit test (`test_game` Meteor's fire with Inferno synergy). Hit recovery trigger (what damage starts hit recovery: d2d `damage * 8 >= max_hp` in whole points, `ai.cpp hurt`) not traced here; the avoid test FUN_0057dfb0 reads the player's avoid/dodge: not relevant to a monster target (U).

## Reproduce
```sh
# The worker host: clone of the user's fork, upstream main af825eda, built with -DFETCHCONTENT_SOURCE_DIR_* as in cast-ticks
export D2_MPQ_DIR=<dir with bin/game.exe and MPQs incl. lowercase patch_d2.mpq>
cd tools/emu && python skill_tips.py 20 0 > g.txt   # (uv run python ...)
cd ../.. && build/tests/test_game --skill-tips 20 0 | grep '^[0-9]' | diff g.txt -      # empty
cp <this folder>/missile_roll.py tools/emu/ && python tools/emu/missile_roll.py 3000    # ok
python tools/emu/disasm.py 5adf10 ; ... 5a89a0 ... 5ad730 ... 57c6c0
```
