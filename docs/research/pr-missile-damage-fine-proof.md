# Proof: a missile's element damage on a monster keeps its 256ths (fix/missile-damage-fine-v2)

> status: current (agent PRPROOF, 2026-10-30). Applies docs/process/upstream-pr-standard.md to the candidate of docs/research/pr-missile-damage-fine-draft.md. Branch `fix/missile-damage-fine-v2` in `~/d2d-lab/pr/gaps` on the worker host, commits `4e6d9d39` (the fix, tests, docs note) and `e8dd9efd` (tools/emu/life_apply.py), rebased on upstream/main `32768ddb`, author and committer the user, no trailer. Nothing pushed or posted. Tags: V = run or read by this agent (command given), S = upstream doc, U = unverified.

## Claim (one)

game.exe applies a missile's element to a monster in 256ths end to end: FUN_005a89a0 rolls `min + rand(max - min)` in 256ths (V, missile_roll.py 3000 cases, docs/research/upstream-gaps/item1-charged-bolt-hit.md), FUN_0057c1e0 folds the record's elements through FUN_0057bf80 (resistance by FUN_0057be00, then MulDiv FUN_00483360 of the fine value, then absorb FUN_0057bf10) into the total at record +0x4c (LAB_0057c3bd), and FUN_0057c6c0 takes that total off the life stat 6 (LAB_0057c852..0057c874): under 0x100 after the hit is dead, set to 0. A heal is FUN_0057a980: 256ths added, capped at max life (FUN_00625d10). d2d cut the roll to whole points first (`roll >> 8`), so a Charged Bolt of 2-4 dealt 2.5 on average, not 3.0.

## The chain, tagged (all V by `disasm.py --exe /tmp/casttrace/game.exe <addr>`, listings in /tmp/prproof/*.txt on the worker host)

| Link | What it shows | Tag |
|---|---|---|
| FUN_005ad730 LAB_005ad8c6 | calls FUN_0057c6c0(target, 1, record): apply flag 1, record result word bit 1 set | V |
| FUN_0057c6c0 LAB_0057c7a0 | with the flag, FUN_0057c1e0(game, attacker, target, record) before the life block | V |
| FUN_0057c1e0 LAB_0057c242 / 0057c24b | "target is a plain monster" = type 1 and not one of FUN_0063ee90's hireling classes (S: merc-ai.md); LAB_0057c3a5..0057c3bd: total +0x4c = +0x08 + +0x10 + +0x1c + +0x20 + +0x24 + +0x28 | V |
| table 0x732980 row 2 | lightning: record +0x1c, resist stat 0x29, max stat 0x2a, pierce stat 0x14e, absorb 0x90 / 0x91 (dumped with emu.r32) | V |
| FUN_0057be00 | res = stat 0x29; pierce taken off when res < 100 or a plain monster; a plain monster gets no difficulty penalty (LAB_0057be42) and no 75 % cap (LAB_0057be8d); floor -100 | V |
| FUN_0057bf80 LAB_0057bfd4..0057bff4 | dmg > 0 and res != 0: res >= 100 becomes 100; dmg = MulDiv(dmg, 100 - res, 100); res 0 leaves dmg as is; flat magic reduction (stat 0x23 << 8) taken off first (LAB_0057bfd0) | V |
| FUN_00483360 | MulDiv: a <= 0x100000 and b <= 0x10000: a * b / c; a > 0x100000 (c <= a >> 4): a / c * b | V |
| FUN_0057bf10 | absorb stats 0x90 (% , at most 40) / 0x91 (flat << 8) heal the target and reduce the damage; 0 for the monsters in the claim | V |
| LAB_0057c852..0057c874 | life = GetStat(6) - total; < 0x100 then 0; SetStat(6) | V |
| FUN_0057a980 | life + amount capped at FUN_00625d10 (max life); 0 or less adds nothing | V |
| stats 6 / 7 in 256ths | S: upstream monsters.md ("stats 6 / 7 in 256ths"), corroborated by the 0x100 death threshold | S+V |

No U link. Not in the chain (hooked in the oracle, listed in life_apply.py's docstring): stat getter/setter (a dict), max life, unit-kind predicates, difficulty row, events, stun / cold / freeze / poison / burn, logging.

## The differential (V)

Oracle `tools/emu/life_apply.py` runs the REAL FUN_0057c6c0 chain and FUN_0057a980 on game.exe (Unicorn, upstream tools/emu) and prints the life (256ths) after every hit or heal; `test_game --life-apply SEED N` runs the REAL d2d `rules::missile_blow` + `hurt()` + `heal()` on the same cases (shared xorshift32 generator) and prints the same lines. Cases: exhaustive boundaries (13 resistances -100..150 x lives 1..4 points x every roll 1..1100 256ths, plus MulDiv's 0x100000 edge at max life 0x7fffff) and N random sequences (max life 1..300, resistance -100..150 incl. immune, pierce 0..60 against a non-immune, 1..12 events of Charged-Bolt-like rolls, boundary rolls, huge rolls and whole-point heals). The script also runs a line-for-line copy of the OLD d2d arithmetic (upstream/main `resisted(roll >> 8, res)`, whole-point life, clamp heal) and of the NEW one, and counts their mismatches against game.exe.

| Scenario | lines / events | old d2d wrong | new d2d (copy) wrong | real d2d binary vs game.exe |
|---|---|---|---|---|
| boundaries, exhaustive (B, M) | 57,304 lines | counted in the total | 0 | diff empty |
| random sequences, seed 1 | 100,000 lines, 500,425 events (with B, M) | 307,385 of 500,425 | 0 | diff empty (157,304 lines) |
| smoke, 200 cases | 57,504 lines, 58,156 events | 30,231 | 0 | diff empty |
| 2,000 cases after the last test edit | md5 e7dae12e… both sides | | 0 | identical |
| 3,000 cases, seed 7, after the rebase on 32768ddb | 70,523 events | 37,993 | 0 | md5 e65c1d2e… both sides |
| 3,000 cases, seed 99, final commit (rolls 0, -1, -256, 0x7fffffff added) | 60,356 lines, 70,698 events | 38,030 | 0 | diff empty |
| verifier: seed 99 and 36,292 variant events (res to -1000, pierce to 1000, 200-hit accumulation) | | | 0 | diff empty / all match |

Outputs on the worker host: /tmp/prproof/g100k.txt (game.exe), /tmp/prproof/d100k.txt (d2d), /tmp/prproof/g100k.err (`500425 events: old d2d wrong on 307385, new d2d wrong on 0`, `diff rc=0 lines g=157304 d=157304`).

## What the patch needed (v2 over PRBUILD's 91a7cc92)

1. Resistance on the fine roll by game.exe's own MulDiv branches: `rules::resisted_fine` (res >= 100: 0; res 0: as is; over 0x100000 the quotient comes first). PRBUILD's `roll * (100 - min(res, 100)) / 100` differed above 0x100000 (4096 points a hit).
2. Dead is life 0 (LAB_0057c86c): `hurt()` sets `hit_points = 0, life_frac = 0` when under one point; before, hit_points went negative. Two thorns asserts in test_game.cpp that relied on a negative value now expect 0.
3. `heal(Monster&, int points)` (FUN_0057a980): whole points, capped at max; the fraction is cleared only when the cap cuts (a heal that lands exactly on max keeps it, as game.exe's 256ths do). The two Zakarum heal sites use it. PRBUILD's model left `life_frac` stale on a capped heal: off by up to 255/256 of a point.
4. The oracle and the `--life-apply` mode (tests/test_game.cpp) as the reviewer-runnable evidence.
5. After the verifier: a roll at or under 0 deals nothing (LAB_0057bf94) and the over-0x100000 product wraps in 32 bits like game.exe's imul (a wrapped result deals nothing).

## Commands (reviewer, own copy)

```sh
python3 tools/pr_guard.py                                   # 0 FAIL, 0 WARN
cmake --build build && ctest --test-dir build --output-on-failure   # test_game needs D2_MPQ_DIR and D2_PATCH_INSTALLER
cd tools/emu && uv run python life_apply.py 100000 1 > g.txt      # stderr: ... new d2d wrong on 0
cd ../.. && D2_MPQ_DIR=... D2_PATCH_INSTALLER=... build/tests/test_game --life-apply 1 100000 | grep -E '^[BMR][0-9]* ' | diff tools/emu/g.txt -   # empty
cd tools/emu && uv run python missile_roll.py 3000           # ok (the roll itself; script in docs/research/upstream-gaps/, not in the PR)
```

## Their checks (V, the worker host, build dir /tmp/prbuild-build, D2D_WERROR ON)

`ninja test_combat test_game` clean; `./tests/test_combat` ok; `test_game` rc 0 with the MPQs and LODPatch_114d.exe; `pr_guard.py upstream/main` 0 FAIL 0 WARN; `presubmit.py --quick` 0 FAIL, 3 WARN (siblings: monster not player / merc / pet / co-op partner; skill not melee / missile; build skipped): answered in the body.

## Residual risks and limits (disclosed in the body)

- Domain: a plain monster (no hireling: no difficulty penalty, no 75 % cap), magic damage reduction (stat 0x23) 0, absorb 0, resistance -100..150, pierce only against a non-immune, whole-point heals. Pierce against an immune: game.exe lets it through for a plain monster (LAB_0057be20), d2d keeps the immunity: a separate gap, candidate issue, not changed here.
- Stats 6 / 7 in 256ths is read from upstream's doc and the 0x100 threshold, not from the stat list code (the getter is hooked).
- `life_frac` is not replicated to co-op clients (replication.hpp: `hit_points` only; the server is authoritative, a client's bar is under a point off). The DoT paths (fight.cpp `dot_acc`) subtract whole points on their own and can leave `hit_points` negative, so dead = 0 holds for `hurt()` only. `Fight::land` drops `blow.frac` when `soak()` absorbed part of a blow's physical (Bone Armor), and the Lightning Enchanted release check (`hit_points < last_hp`) misses a fraction-only hit: all pre-existing whole-point paths, outside the claim.
- Other callers of `hurt()` (melee, pets, thorns, poison) pass whole points as before; the only visible change for them is a dead monster's `hit_points` 0 instead of negative (replicated to co-op clients as 0). `soak()` drops the fraction when the armor pool took part of the blow (physical only; a pure element blow passes it whole).
- Rebased on upstream/main 32768ddb (our #127, no overlapping files); guard, build, tests and the seed-7 differential rerun after it. Fetch again right before the push.
- Scenario state hashes (tools/d2ddet, det.py) were not rerun on a patched lab binary in this job (time box); the fix changes only `hurt`, `heal`, `missile_blow`'s element branch.
- Statistics (PRMEASURE, docs/research/pr-missile-damage-fine-measured.md) are corroboration only: Charged Bolt 8 casts per kill 0.60 -> 0.52; CB 4 within noise; CB 1 disturbed by host load.

## Verifier (independent, second agent, Fable; report verbatim)

VERDICT: holds with limits

What I ran (the worker host, outputs under /tmp/prproof-verify/):
- Read the diff, oracle and listings 57c6c0/57c1e0/57bf80/57be00/483360/57bf10/57a980/63ee90/625d10/5ad730. By hand: MulDiv branches (divide-first above 0x100000, 32-bit imul/idiv below) equal `resisted_fine`; `life < 0x100 -> 0` is exactly `hp<1 || (hp==1 && frac>0)`; heal caps at 625d10 = stat-7 getter. Hooks: 63ee90 returns nonzero only for hireling classes (0x10f/0x152/0x167/0x230-1), so 0 for class 60 is right and the flag at 57c242 means "no difficulty penalty, no 75 cap, no pierce through immunity" in 57be00, consistent with known behaviour. 5ad730 passes applyflag 1, rec flags 1 (no 0x100/0x200/0x400). [Ideas 1, 2, 5 done]
- Fresh seed: `life_apply.py 3000 99` vs `test_game --life-apply 99 3000`: 60,304 lines, diff empty (old model wrong on 38,026 events, new on 0).
- /tmp/prproof-verify/variants.py (36,292 events vs the New model): res -101..-1000, pierce to 1000, heal 0, heal to exactly max, 200-event fraction accumulation, exact 255/256/257 remainders, every res -100..99 at the 0x100000 edges: all match. [Idea 3 done]

Mismatches found, all outside the stated domain:
1. Negative roll (-1, -256): game.exe zeroes the field (57bf94); `resisted_fine` goes negative and hurt() would heal. Only if `rng.range(elo, ehi)` can be < 0 (no real skill); a `roll <= 0` guard is cheap.
2. roll 0x7fffffff: game wraps in 32-bit MulDiv (no damage); d2d `fine/100*(100-res)` is signed-overflow UB. Unrealistic (8M points).
3. Heal on a dead monster: oracle hooks 5541b0 (unit-state check) to 0 so game.exe heals a corpse there; d2d refuses. The oracle cannot adjudicate this; the real guard exists.

Limits, not breaks [Idea 4]: `life_frac` not replicated (components/game/replication.hpp:128), co-op clients lose it; DoT paths (fight.cpp:969, 2962) subtract whole points via their own `dot_acc` and can leave hit_points negative, so "dead = 0" is not universal; Fight::land drops `blow.frac` whenever soak() absorbs any physical (element 256ths vanish on Bone-Armor partial absorbs); LE-release `hit_points < last_hp` (fight.cpp:1716) misses frac-only hits. Not checked: the Zakarum heal amount (`max*pct/100` whole points) against its srvdofunc; FUN_005a89a0's roll range. Branch is based on af825eda while upstream/main is now 32768ddb (the diff's book-price removal is only that): needs a rebase before PR.

**Author's response (V):** findings 1 and 2 fixed in the final commit 4e6d9d39 (`resisted_fine`: nothing at or under 0 per LAB_0057bf94; the over-0x100000 product in 32 bits, a wrapped result deals nothing; boundary rolls 0, -1, -256, 0x7fffffff added to both sides: identical, `h...=8388352` untouched on both). Finding 3 is the oracle's hook, not the claim (d2d's `heal` refuses a corpse; game.exe's FUN_005541b0 guard does the same, hooked to "alive" here). The rebase was done (32768ddb). The four limits are listed under residual risks and in the PR body.
