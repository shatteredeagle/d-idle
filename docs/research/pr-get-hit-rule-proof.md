# Proof: the get-hit (hit recovery) decision matches game.exe 1.14d (agent HITREC, 2026-10-30)

> status: research, current (card hitrec). Standard: memory d2-pr-proof-standard (differential vs game.exe's own code, zero mismatches after, one claim, limits disclosed, adversarial check). Offline only: no game instance, nothing pushed, posted or filed. Tags: V = run in game.exe's own code under unicorn by this agent (`tools/emu/hit_rule.py`) or read in the capstone dump of the read-only copy (`/tmp/casttrace/all.txt` on the worker host); V-d2d = read or run in d2d (upstream main 32768ddb, the patched clone); S = upstream `docs/research/re/*`; U = unverified.

## The claim (one claim, one PR)

When a landed hit (result flag 4) reaches a unit that is not stunned, d2d now decides the get-hit as game.exe's `FUN_0057cb00` does: no get-hit under one point (256ths); divisor d by the record's hit class (jump table `0x57cbf8`: 8 for classes 2, 6, 10, 11; 32 for 4, 8; 64 for 5; 16 for 1, 3, 7, 9, 12 and every other value); under max / d none, under 2 max / d a roll of 2 on the unit's seed (`FUN_00472210`, `& (n - 1)`) of 0 is none, then under 4 max / d a roll of 4 of 0 is none, else a get-hit (bands 37.5 % / 75 % / sure). The class is the record's +0x60 after `FUN_0057ce30` (0x57cf54): 13 for none, and an element's bits (cold 0x30 / fire 0x20 / lightning 0x40 / poison 0x50 when the record field 0x24 / 0x10 / 0x1c / 0x28 is over 0, table `0x6e16d8`) or a doubled hit's 0x10 (record flag 0x2000, set at `0x57b929` when the mastery crit, critical strike or deadly strike roll of `FUN_0057b7d0` lands; magic +0x20 is not in the table) OR-ed in, so an elemental or doubled blow divides by 16 whatever the weapon. d2d before: a fixed twelfth (player fight.cpp:4903, merc :4800) and eighth (monster ai.cpp:2853), no class, no roll.

NOT claimed (left as d2d has it, said in the PR): the length of the get-hit (game.exe rate = AnimData speed x (50 + FHR) / 100 vs d2d's 100 + FHR, DECOMPREC section 3; its server-side monster mode end is still U); a frozen unit's exemption (state 1; d2d has no freeze state, it stuns); the monster's MonStats2 GH-mode bit; the stunned monster's sure get-hit (d2d's stunned monster stands anyway).

## U items from DECOMPMON / DECOMPREC: closed or narrowed (V unless marked)

| Item | Finding | How |
|---|---|---|
| record +0x28 | the poison per tick (record int 10; `0x57c8bc`..`0x57c8c5` hands [0x28], [0x2c] to the poison applier `FUN_0057ac50`, S combat.md "Cold, freeze and poison lengths" agrees); `0x57cb22`: a record whose poison equals its total gets no get-hit | read |
| hit class source | `0x57cf41`..`0x57cf5f`: with result flag 1, record +0x64 == 0 and no 0xf0 bits in +0x60, `FUN_0057ce30` rewrites +0x60 (element bits, 0x10 for flag 0x2000, 13 for 0) and copies it to unit +0xb0; both callers (`0x57d0c2` monster, `0x57d374` player) pass that +0x60. For missiles +0x60 is Missiles.txt HitClass (Charged Bolt 64 -> 16); for melee the weapon's / MonStats2's class (S, d2d already carries both as `hit_class`) | read + run (the oracle runs `FUN_0057cee0` whole) |
| flag 0x2000 | a doubled hit: the mastery crit (`0x57b8b4`), critical strike (stat 0x151, `0x57b8e0`) or deadly strike (stat 0x8d, `0x57b903`) roll landed (`0x57b924`..`0x57b92f`); d2d's `blow.deadly` is the same three rolls | read (verifier) |
| record +0x64 | a byte that, when set, keeps the class as is; what sets it is U (writers found only in 0x59xxxx client code). d2d has no such field; the oracle randomises it and the harness models it, so the function is proven for both values | narrowed |
| callers | upstream `game-calls.tsv`: `FUN_0057cee0` is the only caller of `FUN_0057cb00`; stun (state 0x15, `0x57d0b6` / `0x57d368`) skips it and gets the hit anyway | read (S for the tsv) |
| monster mode end (-1 tick) | untouched: it belongs to the length claim, not this PR | narrowed |

## Differential (V): game.exe's own `FUN_0057cee0` vs d2d

Oracle `tools/emu/hit_rule.py` runs `FUN_0057cee0(unit, record)` natively for each case (hooks: the room 0, state tests from the case, max life from the case, `FUN_0046c140` bit 3 from the case, the mode calls `FUN_005809d0` / `FUN_005a7e60` / `FUN_0064c040` recorded; the seed roll runs natively on unit +0x20) and reports a get-hit when mode 4 (player) or 3 (monster) was set; a case that records neither or both aborts. Cases: 2264 boundary (every tier edge +-1 for classes 0..13 and 0x40 / 0x20 / 0x43 / 0x12, both kinds, five max lives; pure poison, frozen, stunned, no-GH-mode, elemental and deadly at four edges) + 100,000 random (max life 1..3000 points with low bits, damage spread over the bands, classes 0..15 and element / missile values, +0x64, elements, poison-only, flag 0x2000, GH bit, stun, frozen, seeds), seed 7.

| Side | Code | Cases | Differ | Player | Monster |
|---|---|---|---|---|---|
| BEFORE | `tools/emu/hit_rule_harness.cpp` (quotes fight.cpp:4903 / ai.cpp:2853; a stunned player sure; poison-only never reaches either) | 102,264 | **19,343** | 8,701 of 50,863 | 10,642 of 51,401 |
| AFTER | upstream's own `tests/test_combat --get-hit` on the patched clone (calls `rules::get_hit`; stun / frozen / poison-only / GH bit modelled around it as the real paths do or do not, see limits) | 102,264 | **0** | 0 | 0 |

Checksums on the worker host: cases `6198b118`, game `3dd7cb3b`, before `1024c42b`, after `249d5e9e` (md5 prefixes of `/tmp/hitrec-*.tsv`).

```sh
# The worker host; upstream harness in the clone, read-only game.exe copy, never run as a game
cd ~/d2d-lab/pr/hitrec/tools/emu && cp ~/code/d2lod-wt/hitrec/tools/emu/hit_rule.py . && source /tmp/casttrace-venv/bin/activate
python hit_rule.py gen 100000 7 > /tmp/hitrec-cases.tsv
D2_MPQ_DIR=~/d2d-lab/data python hit_rule.py --exe /tmp/casttrace/game.exe game /tmp/hitrec-cases.tsv > /tmp/hitrec-game.tsv   # 16 s
g++ -O2 -std=c++20 -o /tmp/hit_rule_harness ~/code/d2lod-wt/hitrec/tools/emu/hit_rule_harness.cpp   # (copy it over first)
/tmp/hit_rule_harness < /tmp/hitrec-cases.tsv > /tmp/hitrec-before.tsv
/tmp/hitrec-build/tests/test_combat --get-hit < /tmp/hitrec-cases.tsv > /tmp/hitrec-after.tsv
python hit_rule.py compare /tmp/hitrec-game.tsv /tmp/hitrec-before.tsv /tmp/hitrec-cases.tsv   # 19343 differ
python hit_rule.py compare /tmp/hitrec-game.tsv /tmp/hitrec-after.tsv /tmp/hitrec-cases.tsv    # 0 differ
```

## The patch (V-d2d)

`~/d2d-lab/pr/hitrec` (fresh clone of the user's fork; `upstream` = psi29a/d2decomped, main 32768ddb), branch `fix/get-hit-rule`, one commit `d55973e1` (amended after the verifier's report) "Get-hit by hit class and share of life", author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`, no trailer. 7 files, +120 -9: `rules::get_hit_divisor` / `rules::get_hit` and `Blow::elemental` (combat.hpp; set in `player_blow` and `missile_blow`); `hurt(.., hit_class, marked)` (ai.hpp / ai.cpp, the roll on `monster.seed`); `Fight::land` passes `blow.hit_class, blow.elemental || blow.deadly`; the player's struck frame uses `foe.hit_class, foe.elemental > 0` with d2d's `rng`; the merc the same with `merc->seed`; combat.md gets the section "Hit recovery: the get-hit test" and loses the ponytail line; `test_combat` gets the asserts and `--get-hit`; `test_game`'s arrow scenario deals max / 2 instead of max / 4 (a quarter is only the 75 % band for the arrow's class 10 / 11; a half is sure for every class).

Checks on the worker host (`/tmp/hitrec-build`, Release, FETCHCONTENT from `~/d2d-lab/src/build/_deps`): `ninja test_combat test_game` ok; `./tests/test_combat` -> `ok`; `D2_MPQ_DIR=~/d2d-lab/data D2_PATCH_INSTALLER=~/Downloads/LODPatch_114d.exe ./tests/test_game` -> 23 `OK` lines, rc 0 (it FAILED before the scenario fix at test_game.cpp:551: the quarter was no longer sure); `python3 tools/pr_guard.py upstream/main` 0 FAIL 0 WARN; `python3 tools/presubmit.py --quick` 0 FAIL, 3 WARN (siblings: merc / pet / co-op; missile / skill paths; build skipped) answered in the draft body.

## Residual risks and limits (say them in the PR)

1. d2d compares this frame's whole points of damage (`foe.damage`, `hurt`'s `damage`) << 8 against max life in 256ths; game.exe compares the record's 256ths. Equal at whole points; a blow's fraction (d2d rounds monster blows to whole points already) can sit on the other side of an edge.
2. Player: the rolls take d2d's game `rng`, not the player's own unit seed (marked ponytail, as the other player rolls are); the merc and monsters roll on their own `seed` (as game.exe).
3. game.exe's size test runs on the record total +0x4c, which sums the poison field [0x28] (and magic +0x20) with the rest (`0x57c3a5`..`0x57c3c7`, verifier); d2d tests the blow's damage without its poison (poison runs over ticks). A physical + poison blow can be a roll in game.exe and none in d2d. Not modelled by the harness (it takes the total as given).
4. Not ported: the frozen exemption, the MonStats2 GH-mode bit, the stunned monster's sure get-hit, the town / attacker gate at `0x57cef4`..`0x57cf37` and the state 0x41 / 0x42 / 0x44 and flag 0x80 / 0x100 / 0x200 / 0x8010 branches that take precedence over flag 4 in the player branch; thorns / fire blast / pet / monster-on-monster callers of `hurt` pass class 0 (16) since d2d has no record there (game.exe would compute 13 -> 16 for a plain record, U for those sources).
5. The record's +0x64 byte (keeps the class) is U; treated as 0 in d2d.
6. No live or state-hash run: `tools/d2ddet/pair.sh` compares two runs of one binary (determinism), not before / after; the corroboration probe (`cmd/d2dcasts2`) needs the lab fork's debug verbs, not upstream's tree. Both left undone in the time box; the regression evidence is upstream's scripted `test_game` (23 scenarios) and `test_combat`.
7. The arrow scenario in `test_game` was changed (max / 4 -> max / 2); a reviewer may ask why, the answer is in the test's comment.

8. The player's test draws 0 to 2 rolls from d2d's shared game rng (not the player's own seed, which d2d does not keep): later rolls in the same frame shift against the old build (replay / lockstep: the same on every peer running `hurt_side` identically).

## Round 2: PR #134 review (CodeRabbit), the get-hit per landed blow (V)

CodeRabbit's Major finding (fight.cpp:4800 / :4907: the frame's summed damage tested once) is VALID: game.exe tests one damage record at a time (`FUN_0057c6c0` then `FUN_0057cee0` per hit: `0x56ba73` / `0x56ba7f`, `0x57d5c7` / `0x57d658`, `0x5a6e82` / `0x5a6e90`, `0x5ad8c6` / `0x5ad8d3`). The oracle now emits multi-record frames (`next` column: the following record lands on the same unit, the seed carrying on; one result a frame) and the differential was re-run (seed 7, 73,694 frames, 21,532 with 2 or 3 records):

| Side | Differ | Of which single-record frames |
|---|---|---|
| upstream main (twelfth / eighth on the sum) | 13,935 | - |
| d55973e1 (get_hit on the sum, gates modelled) | **702** (all player, all multi-record) | 0 of 52,162 |
| e3c1e535 (per blow: `Foe::blows`, `struck_big`) | **0** | 0 |

Fix commit `e3c1e535` on the same branch (author shatteredeagle, no trailer; +56 -22, amended with the absorb-order and burning-ground fixes after the second verifier round; test_combat ok, test_game 23 OK, pr_guard 0 / 0). Replies and push commands: docs/research/pr-get-hit-rule-review-draft.md. Checksums: cases2 `27625643`, game2 `e7e49d38`, after2 `4fdb5ced`, before2-main `0afd13c4`, before2-sum `f3712806`. Second verifier round: HOLDS WITH LIMITS; its two fixes are in e3c1e535, the rest is disclosed in pr-get-hit-rule-review-draft.md (aura blow is no `hit` for the stun rule; thorns / poison ticks have no blow; the mid-tick mode change; the room gate). After the amend: test_combat ok, test_game 23 OK, 0 differ on the 73,694 frames, pr_guard 0 / 0.

## Verifier verdict (independent adversarial agent, general-purpose, Fable, 7 min, read-only)

**CLAIM HOLDS WITH LIMITS.** It re-read `0x570209`..`0x571075` and `FUN_0057ce30` by hand (table decoded from bytes, both callers, mode calls), re-ran the oracle on a fresh `gen 100000 424242` (102,264 cases, 0 differ) and 34 adversarial cases of its own (0x7fffffff and negative max life / damage, max life 0 and 1, damage over max, classes -1 / -6 / 13 / 14 / 258 / 4096 / 65540, +0x64 with class 0, poison over / equal with stun, stun + frozen, no GH bit + stun: 0 differ), and checked that the hooked-out `FUN_005a54f0` writes none of the record fields used. Its findings, and what was done: (1) flag 0x2000 is set by three doubling rolls, not deadly strike only: wording fixed in code and docs (d2d's `blow.deadly` already matched); (2) +0x4c sums the poison field: disclosed as limit 3; (3) magic damage is not in `FUN_0057ce30`'s table but d2d marked it: fixed (`marking` excludes element 4 in `player_blow`, `etype != 4` in `missile_blow`); (4) a monster's poison did not mark the player's / merc's blow: fixed (`|| foe.poison > 0`); (5) runtime gaps (stunned monster, frozen, poison-only, GH bit, class 0 for thorns / blast / pets / monster-on-monster; knockback ordering) and (7) the town gate and the player-branch precedence: disclosed as limit 4; (6) the shared rng draws: limit 8. Not checked by it: `FUN_00625d10` returning stat 7 in 256ths for monsters, where DR / absorb is applied before +0x4c, `FUN_00623c20`'s class for monster attackers (`0x57cea0`, a path the oracle starts after). After its fixes the differential was re-run: 0 differ (`/tmp/hitrec-after.tsv` md5 unchanged in count), test_combat ok, test_game 23 OK.

## Mark derivation (merge-resolution check, agent MARKPROOF, 2026-10-31)

Question: after the merge with #135, `missile_blow` sets `Blow::elemental` from the fine value. The decision differential (`hit_rule.py`) takes `marked` as input, so the derivation of the mark from the damage record was unchecked.

game.exe 1.14d (V, read in the disassembly and run natively): `FUN_005ad730` calls `FUN_0057c6c0` (-> `FUN_0057c1e0`, a loop over table 0x732980 -> `FUN_0057bf80`, which resists the field and WRITES it back; a field at or under 0 becomes 0) and then `FUN_0057cee0` on the same record, whose `FUN_0057ce30` (0x57ce30) walks four entries of table 0x6e16d8 (cold +0x24, fire +0x10, lightning +0x1c, poison +0x28) and takes the first whose field is over 0 (signed compare, 256ths: 1 counts; magic +0x20 and physical +0x08 are not there). The start index rotates with a byte counter at 0x88cad0, so with several fields the class bits differ run to run; the mark (any field over 0) does not. It is per record = per blow, after resistance.

Oracle `tools/emu/hit_mark.py` (+ `hit_mark_harness.cpp`): game.exe runs `FUN_0057c6c0` then `FUN_0057cee0` per case and the class at unit +0xb0 is read (13 = none); the harness calls the real `rules::missile_blow` of the merged tree. 111,200 cases (boundary: every element x rolls 0, 1, 2, 127, 128, 255, 256, 257, 511..513, 0x100000 +-1, 0x200000, 0x7fffffff x resistance -100..150 x pierce x length; min < max with a physical draw; 100,000 random, seed 7). Merged tree 6fc9451e: fire, lightning, cold, magic and stun 0 differ; poison 3,923 differ (a total under one point: `roll * length >> 8` is 0 while the per-tick field is over 0). One-line fix in `missile_blow` (`resisted_fine(roll, res) > 0` for the poison blow; the totals are unchanged): 0 of 111,200. `hit_mark.py multi` (20,000 records with several fields set, game.exe only) matches the any-of-four model, 0 differ.

Tags: V for the rule and the differential; UNVERIFIED that the missile hit fills the poison field with the roll per tick (the fill is not traced; `FUN_005a89a0` rolls it, the evidence pack of #134 calls +0x28 per tick). Domain: plain monster, no magic damage reduction or absorb, physical resistance 0, pierce on a non-immune only.
