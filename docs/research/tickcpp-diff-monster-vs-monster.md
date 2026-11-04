# Monster hits monster: does game.exe keep the 256ths? (job tickcppdiff, 2026-11-04)

> status: PARTIAL, no number run. Tags: V read = read in game.exe's disassembly (`tools/emu/w3_dis.py`, run on the worker host); U = unverified. There is no V run, no case count and no verdict on a differing count in this document. CodeRabbit's finding (PR 189: `monster_blow` returns the 256ths, the monster-vs-monster blow at `fight/tick.cpp` 344-345 drops them) stays OPEN: it cannot be closed with proof from here, and it cannot be confirmed as a deviation either.

## What was denied or missing

- The d2d tree after batch B (`fight/tick.cpp`, lines 344-345) is not readable from this runner: the readable upstream clone on the worker host is older (no `components/game/fight/` directory, `monster_blow` still has the whole-point callers in `ai.cpp`), and no read-only checkout of the merged tree was reachable. So what d2d does at that site is U (the finding's own text is the only source).
- The oracle `tools/emu/monster_damage.py` of batch B (named in `pr-batch-b-body-r5.md`) is in neither this repo nor the readable clone. The older harness `tools/emu/v4_run.py` / `w3_run.py` run `FUN_0057b420` natively but model `FUN_0057c1e0` themselves, and always with a player-like defender. A monster defender run needs a new harness (below).
- One compound shell command (a `grep -E` after the runner) was denied by the allow-list; not retried in another spelling.

## Q1 What game.exe does when the defender is a monster (V read)

`tools/emu/run-on-worker.sh w3_dis.py 57c1e0:260 57c6c0:140`:

| Address | Reading | Tag |
|---|---|---|
| `0057c6c0` (apply damage, args: defender `[ebp+8]`, flag `[ebp+0xc]`, record `[ebp+0x10]`, attacker in EDX, EDX/ECX saved) | The defender is any unit: `0057c6c8 mov esi,[ebp+8]`; its type is read as `[esi]` (`0057c760`), with 6 for a null unit (`0057c757`). A monster defender (type 1) only adds a gate: `0057c76a-0057c777` calls `0x457490` on the monster record and skips the damage when it says so (untargetable or damage-immune class, not decoded: U). Everything after is the same code as for a player. | V read |
| `0057c793-0057c7a0` | `if flag != 0: call 0057c1e0` (the record reduction), same call for every defender type | V read |
| `0057c7b1-0057c7f3` | absorb events (`0x5c0c30`, events 5/6 then 1/2) raised on the defender, whatever its type | V read |
| `0057c7f8-0057c826` | the life stat (stat 6, read `0057c803 / 0057c818`) is compared with the record's life field `[edi+8]`, the record is stored and `0057c420` applies it. Stat 6 is in 256ths for every unit type (it is the same stat on a player and a monster). | V read for the code, I for "256ths on monsters" (the unit of stat 6 on a monster was not measured) |
| `0057c1e0` (reduction) | Both units are tested for type 1 and passed to `0x63ee90` (`0057c214-0057c23b`); the defender, if a monster, goes through `0x63e990` / `0x63e940` (undead / demon flags) and the record flags `0x100` / `0x200` / `0x400` (`0057c317-0057c367`). So a monster defender is handled explicitly, with the same reduction loop (`0x57c378`, `FUN_0057bf80`) as a player. | V read |
| Roll | `FUN_0057b420` takes the attacker only (earlier traces); nothing in `0057c6c0` or `0057c1e0` branches on the defender being a player before the roll. Which caller builds the record when a pet, merc or converted monster is the attacker was NOT read. | U |

So (V read) the apply-damage function does not truncate the record for a monster defender: if the record carries 256ths, the 256ths reach the defender's life. Whether the record's roll is in 256ths when the attacker is a monster striking a monster depends on the attack function that calls it: not read (U). By the player-defender result in `monster-attack-trace.md` (`0057ec10` family, `0057b420` roll) the monster attack function rolls in 256ths regardless of the target; that is S for a player target, I for a monster target.

## Verdict (honest)

- game.exe: V read that the shared apply path keeps whatever fraction the record has, and gives a monster defender the same absorb and reduction; I that the monster attacker rolls in 256ths for a monster target.
- d2d: U (tick.cpp not readable).
- Differing count, first differing case: none (nothing run). Not a verdict V (ran), not a closed finding.

## Deviation candidate (not a claim)

D1 (U): `fight/tick.cpp:344-345` drops `taken.frac` for a monster hit by a monster, while game.exe keeps it in the defender's life (stat 6, 256ths). Expected effect if true: the defender (pet, merc, converted monster, or a monster hit by a pet) keeps a little more life per blow than in game.exe (under one hit point per blow). The size is what the run must give; none is quoted.

## Ready-to-run (needs the merged d2d tree and a new harness)

1. Harness `tools/emu/tickcpp_mvm.py` (to write): copy `w3_run.py`; make the defender unit a monster (type 1, a record from `monstats` with its own stat list: life stat 6, resist stats 0x24 / 0x25, no absorb states), call game.exe `FUN_0057b420` natively with a monster as attacker (EDX = 1, stats 21 / 22 hooked as in `w3_run.py`), then `FUN_0057c6c0` itself (not a model) with flag 1 on the defender so `0057c1e0` really runs; record the defender's stat 6 before and after and the attacker's seed after. Oracle: the real code, byte for byte.
2. d2d side: a probe built like `v4_probe.cpp` that calls the merged `rules::monster_blow` and then the exact statements of `tick.cpp:340-350` on a `Monster` defender (copy the lines into the probe; do not edit the fork).
3. Run: `tools/emu/run-on-worker.sh --ttl 3000 tickcpp_mvm.py 10000 20261104` and again with seed `20261105` (20,000 cases, two seeds); classes: plain, with percent reduction, with flat reduction, 0..0 and X..X records (the `exact` classes of `verify-dmgroll3.md`). Count: life differs, seed after differs, first differing case.
4. If life differs and the seed matches: the fix is one line carrying `taken.frac` into the defender's life at the site, its own small PR (one subsystem: monster-vs-monster blow), test = the probe as `test_combat --monster-vs-monster`, risk = pets and mercs lose slightly more life (det hashes of any scenario with pets or converted monsters change; run the det set). Commits: (1) the carry, (2) the test. No PR candidate until the differential shows mismatches before > 0 and 0 after with the existing harness.

## Drill recipe (lab binary, U items)

Confirm in the lab (not run here): a Necromancer skeleton hitting a Fallen with a known max life, read the life stat before and after one blow with the fork verbs of `docs/forkdbg.md`; a fractional stat 6 after a blow means the 256ths are kept. Repeat for a charmed (converted) monster, per `docs/forkcheats.md`.

## Bot findings

none (no bot code involved).

## Matrix-cell update

No matrix row changes proposed (nothing proved).

## Shared-file lines for the merger

- BACKLOG: `tickcppdiff` PARTIAL: monster-vs-monster 256ths (tick.cpp 344-345) read in game.exe (shared apply path keeps the fraction), no run; needs merged tree + harness `tickcpp_mvm.py`; see docs/research/tickcpp-diff-monster-vs-monster.md
- INDEX: `docs/research/tickcpp-diff-monster-vs-monster.md` - monster hits monster, 256ths (partial, read only)
