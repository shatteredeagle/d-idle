# Fidelity fixes against game.exe 1.14d: vendor repair gate, melee blow, freeze rules, standing missiles, Smite and block

> status: FINAL BODY, not posted (card bundlefinal2, 2026-11-02). Nothing is pushed to the public fork and no PR is open: the coordinator opens it. Branch `local/bundle-final` on the private fork remote, head `48faf360`: 11 commits on upstream main `bdc1eb02` (PR 176 merged). Replaces `local/bundle-4` and `local/smite-block-v1-final`. Tags: V = run (how stated), S = secondary (an earlier run of ours, not re-run for this bundle), U = unverified.

---

## Summary

Six small fixes where d2d differs from game.exe 1.14d, eleven commits, one purpose per commit, on current upstream main. Each has a differential: game.exe's own functions run natively under the emulator in `tools/emu`, the real d2d build runs the same generated cases, and the two are compared. Size: 21 files, +833 / -51 (`git diff --shortstat upstream/main...HEAD`).

| # | component | commits |
|---|---|---|
| 1 | Vendor repair gate | 1 |
| 2 | Melee blow keeps its 256ths | 1 |
| 3 | Undead / Blunt bonus on a player blow | 1 |
| 4 | Freeze rules (Glacial Spike resist and divide, special monsters chilled, Blood Raven and the Countess not halved) plus the harness | 4 |
| 5 | Standing missile footprint and one burst roll, plus the harness | 3 |
| 6 | A block no longer stops Smite | 1 |

Rebased onto the merge of PR 176 (batch 9): one text conflict, in `repair_cost` (`components/rules/rules.hpp`). PR 176 changed `indestructible(item)` to `indestructible(tables, item)` in the line the repair commit rewrites; the repair block is kept and its `durable` line calls `indestructible(tables, item)`. PR 176's comment on stat 0x76 in `missile_blow` still reads correctly after the freeze commits (V: read; no monster gets that stat).

## 1. Vendor repair gate

game.exe's repair price (`FUN_0062efb0`) asks `FUN_0062e660` first: no price unless the ItemTypes row has Repair or the base has a durability (`FUN_00629930`). d2d priced a drawn-durability bow below its max. Counted against game.exe: 7,908 gate cases (659 bases x 12 states) and 60,000 price cases (S, earlier independent run). "Something to repair" where d2d prices and the game refuses: 143 on main, 0 on the head; "whole item": 17 to 0; the 626 whole-item cases where the game prices 1 and d2d 0 are the Repair All sentinel and are unchanged. Multiplier cases above 0xffff: 0 differ on both. With charges missing the gate passes on them alone, and game.exe then scales the base by the durability only when `FUN_00629930` holds (`0062f9e7`): an item without durability pays its whole base, a durable one its missing share. d2d now does the same (a review finding on an earlier version); counted with the same oracle, the base part (the durability scaling, then the npc mult block), 3,000 cases over every non-stacking base with charges missing (V: 2,202 priced): 1,527 differ before that fix, 0 after. Limits: no magic stat costs or sockets in the comparison; stacking items were not in the 3,000 (their restock path is unchanged). In vanilla data all 186 bases of the 66 types without Repair have nodurability 1, so the type half only matters for mods. Re-run on the final head (V, worker host, game.exe's `FUN_0062e660` and the price block under the emulator, seed 2026): gate cases that need repair 4,664 of 4,664 agree, whole items 2,618 agree and the 626 Repair All sentinels differ as before; the base part 3,000 cases (1,963 without durability, 1,037 scaled) 0 differ; the price multiplier cases keep the same 8 buy and 10 repair residuals as upstream main (a minimum-1 rule the comparator does not model).

## 2. Melee blow keeps its 256ths

d2d floored the physical of a player blow to whole points before the resistance; game.exe keeps 256ths (`FUN_0057b420`, `FUN_0057bf80`) and the life stat takes the carry. Counted on landed blows with random weapons, enhanced damage, flat damage and attribute bonuses (S, earlier independent run): 80,250 of 80,557 weapon cases differ in 256ths on main (seed 7), 0 on the head. A second generator (seeds 23 and 5, resistance 0 / 25 / -50 / 75): about 24,000 of 24,040 differ on main in every run, 0 on the head for armed blows with gear percent of -90 or more. Re-run on the final head (V): generator A (`sk_melee2.py`, seed 7, 30,000 cases) 24,088 armed damage cases compared, 0 differ, and every undead cell with a weapon or a Blunt weapon 0 of about 2,300 (barehanded cells still differ: the limit below); generator B (independent harness, seeds 23 and 5, 50,000 each) armed blows with gear percent of -90 or more: 0 and 0 differ, barehanded 7,402 and 7,425 and gear under -90 2,749 and 2,777 still differ. Limits: barehanded blows and the -90 floor applied twice are unchanged and still differ; kicks and Smite share the code but were not counted (U).

## 3. Undead / Blunt bonus on a player blow

Against a MonStats undead (lUndead or hUndead) `FUN_0057b660` adds 50 for a Blunt weapon plus stat 122 to the enhanced damage, only when the sum is over 0, for a player attacker or alignment 2. d2d had no undead input. Counted whole points on undead targets (S): 38,402 differ on main over two seeds, 0 on the head. With components 2 and 3 together the second generator shows 0 armed, gear >= -90 differences in 256ths in all eight runs. Limits (U): whether a hired merc or a Shadow has alignment 2 is read from the disassembly, not run live; stat 122 on a shield is summed with the rest of the gear.

## 4. Freeze rules

`FUN_0057b230` resists and divides a Glacial Spike freeze, chills (never freezes) a special monster, and does not halve Blood Raven's and the Countess's freezes. `freeze_rules.py` runs `FUN_0057c6c0` natively; `test_freeze_rules --freeze-rules` runs the real `Fight::strike`. Author's run (S): 225,748 cases (125,748 boundary cases over every boss kind, resistance -300..150, freeze length -5..250, difficulty, plus 100,000 random): 185,746 differ on main (with the test file), 0 on the head. An independent verifier re-ran it with a new seed on an earlier version of this bundle (V, 155,748 cases, 125,748 boundary plus 30,000 random): 129,904 differ on the base, 0 on the head, with the burst roll and the footprint of component 5 in. Re-run on the final head with a new seed (V, `freeze_rules.py gen 30000 7717`: 125,748 boundary plus 30,000 random cases): 155,748 of 155,748 equal, 0 mismatches. Limits: a freeze length above 0x100000 differs in a separate harness (6 of 16 cases; unreachable in play, S). `Monster::half_freeze` is still set but nothing reads it any more.

## 5. Standing missile footprint and one burst roll

**Footprint.** A standing missile tests its own subtile against the monster each frame (`FUN_005ae1f0`, `FUN_00641cb0`). In game.exe a Size-2 shard hits a size 1 / 2 / 3 monster on 5 / 13 / 21 subtiles and a Size-1 flame on 1 / 5 / 9; d2d hit 21 subtiles for every size. `standing_struck` now uses `missile_touches`, as a moving missile does, for every standing row with a Size of 1 to 3: that is not only the Blizzard shard and Fire Wall flame the tests name, but every such row (Meteor's fire, Blaze, Immolation Arrow's fire, Eruption's cracks, the poison and plague clouds, Shock Field, Volcano and more; 189 rows with velocity 0 and a Size of 1 to 3 in Missiles.txt). A row without a Size keeps the half cell.

Counted (V, an independent harness, a 9 x 9 offset grid, six rows, 8 monster classes of size 1 to 3, game.exe's `FUN_0059fa30` and its own server step against the real d2d `World`): 3,726 hit-area cases; base differs from game.exe in 476, the bundle head in 0. Per row, cases / base differ / head differ: Blizzard shard 648 / 72 / 0, Fire Wall 648 / 132 / 0, Meteor fire 648 / 72 / 0, Blaze 648 / 132 / 0, Eruption crack 648 / 28 / 0 (base misses the 5 x 5 corners of a size-3 monster in 4 of them), poison explosion cloud 486 / 40 / 0. Four of the six rows are not named by the commits. Author's own run (S): 225 offsets x 3 classes, shard: offsets only d2d hits 16 / 8 / 0 for size 1 / 2 / 3 (game.exe 5 / 13 / 21 of d2d's 21); Fire Wall 20 / 16 / 12 (game.exe 1 / 5 / 9). Every head set equals `rules::missile_touches` (V).

**Burst roll.** `FUN_005a9a70` (hit function 1, Fire Ball) and `FUN_005aa8b0` (13, Glacial Spike) call `FUN_005a8c70` once and hand a copy of that record to every unit; each unit resists its own copy. d2d drew a roll per monster. `missile_blow` takes an optional `MissileRoll` that the first blow fills and the others reuse; `area` passes one for hit functions 1 and 13 only. Counted (V, three monsters within a subtile, 60 bursts per spell on the game.exe side, 400 trials on the d2d side): game.exe makes one record and one element roll in 120 of 120 bursts (60 Fire Ball, 60 Glacial Spike); d2d on main gives three equal losses in 2 of 400 trials for each spell (chance ties), on the head in 400 of 400.

Limits: Meteor (hit function 14) also rolls once in game.exe and still rolls per monster here: not fixed. Exploding and Freezing Arrow explosions now share the element roll; a weapon share (`srcdam > 0`) of a shared burst would still be drawn per unit, and whether game.exe copies it is not measured (U). The per-frame hit count of a standing missile is not part of the claim, only the hit set. A monster off its subtile centre (d2d positions are floats), the shrine-potion and plain-missile paths of the same loop, and Size-0 rows are not measured (U). The shared roll shifts later draws against main (a determinism change, expected). Re-run on the final head (V): footprint, game.exe against the real d2d for the Blizzard shard and the Fire Wall flame, 900 offset cases each over 4 classes, 0 differ; burst roll 400 of 400 trials for Fire Ball and for Glacial Spike, game.exe one record in 400, d2d three equal losses in 400. `sk_ground_roll.py cmp` now fails on a d2d line without exactly three positive losses (review finding: `R 0 123` or `R 0 0 0 0` made `same` true; the test asserts the losses are positive before it prints them); proven on constructed files, the old comparison passes them and the new one exits 1, and the real 400-trial runs are unchanged.

## 6. A block no longer stops Smite

`FUN_005ce9f0` (Smite, srvdofunc 150) calls the to-hit / defense check `FUN_0057ec10` at `0x5ceba8` and then ORs the hit bit back in (`or ax, 1` at `0x5cebb0`), so a blocked Smite still deals its damage, its stun and its knockback. `player_blow` ended the blow on any block, so a monster that blocked took nothing. The block roll is made as before (the same RNG draws for every skill and block chance) and marks the blow `blocked`; only a Smite continues to the damage, and `Fight::land` ends the blow on `blocked && !hit`, which is every other swing as before. Of the 33 call sites of `FUN_0057ec10` only `0x5cebb0` ORs bit 0 (Zeal, Sacrifice and the player's Charge are stopped by a block; Blessed Hammer is a missile).

Counted (V, game.exe's own `FUN_005ce9f0` with its to-hit, defense and damage helpers under the emulator, the same cases on both sides; a case lands when the damage reaches `FUN_0057c6c0`; re-run on the final head against upstream main `bdc1eb02`): 30,035 monster cases over seeds 7331 (15,027) and 99 (15,008). Base differs in 3,215 and 3,220, exactly the cases the game blocked; head 0 and 0. Landed damage is inside the game's range in every case, base and head. By block chance 0 / 25 / 50 / 100 percent (d2d's real `player_blow`, 20,000 Smite blows each): head lands 20,000 at every chance, base 20,000 / 15,075 / 10,009 / 0; plain swings are identical on base and head (a plain swing against block 100 is never a hit). Counterfactual in the game itself (the 4 bytes at `0x5cebb0` NOPed): 1,552 of 20,000 land, so the forced hit is what makes the other 18,448 land. New tests in `tests/test_combat.cpp`; the first assertion fails on the base (S: the earlier run of the head test against the base headers, not repeated).

Limits: the block animation is not implemented. After a blocked Smite game.exe puts a monster in block mode and skips its get-hit roll (`FUN_0057cee0`, read from the disassembly, U beyond that); here it shows the ordinary get-hit, because skipping the roll changes that monster's seed stream and d2d's Smite has no to-hit roll before the block roll, so d2d marks a Smite blocked more often than the game does. Player defenders are not modelled (the change is for monster defenders). Quirk documented, not fixed: when a shield's min equals its max the game still rolls up to +255/256ths (10-10 at +100 percent: game 20..21, d2d 20), at most one point; separate from this change.

## Credits

- Vendor repair gate: traced by Claude Code; written by the PR author; independently verified by Claude Code (card verifyprices, independent run).
- Melee fine physical (D1) and undead / Blunt bonus (D2): traced by Claude Code; written by the PR author; independently verified by Claude Code (card verifyskmelee3, independent run).
- Freeze rules: traced by Claude Code; written by the PR author; independently verified by Claude Code (card verifyfreeze, independent run; freeze re-run in card verifybundle).
- Standing missile footprint and burst roll (ground): traced by Claude Code; written by the PR author; independently verified by Claude Code (card verifybundle, own harness).
- A block no longer stops Smite: traced by chuckado (the forced hit bit at game.exe `0x5cebb0`; the commit ends with `Traced-by: chuckado`) and Claude Code; written by the PR author; independently verified by a second Claude Code run (Opus; card verifysmite, own harness, seeds 7331, 99 and 20261102).

Only the git author and the Reported-by / Traced-by lines count for the gap map; verifiers are named here only.

## Checks

All on the final head `48faf360` unless said (V, worker host, Release build with the 1.14d game data, `nice` 19, two jobs).

- Commits: 11, subject line plus a body of at most three lines, then `Agent: Claude Code`; the Smite commit ends with `Traced-by: chuckado`; author and committer the numeric noreply identity on all 11, no co-author or AI lines (V: `git log --format='%an <%ae> | %cn <%ce>'`, `tools/presubmit.py --quick` attribution check ok).
- Suites: test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat (with the Smite tests), test_game, test_sk_ground: rc 0 each. `test_freeze_rules` ran on 155,748 cases (section 4): 0 differ.
- Differentials (counts in the sections above): freeze 155,748 / 0; repair gate and base part, melee generators A and B, footprint, burst roll and Smite all as in sections 1 to 6.
- `tools/pr_guard.py`: 0 FAIL, 0 WARN on the range and on each of the 11 commits. `tools/presubmit.py --quick`: 0 FAIL, 3 WARN, answered below (siblings, doc-claims, build skipped by `--quick`); ruff on the three changed Python files ok.
- g++ and clang++ with the CI flags (`-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror`) plus `-Wshadow -Wrange-loop-construct -Wconversion`, `-fsyntax-only` on all nine changed translation units: g++ 0 warnings; clang++ has no warning on a line the diff adds (the warnings it prints are in untouched code).
- clang-tidy on the nine changed translation units: 0 diagnostics in the 15 changed C++ files; the only ones printed are the `x` / `y` binding names in `components/drlg/outdoor.hpp`, `components/rules/merc.hpp` and `pets.hpp`, which the diff does not touch.
- Size against CodeRabbit's limits: 21 files (limit 40), 884 added plus deleted lines (limit 1000), 11 commits (limit 15).
- Merge: the head is a fast-forward of upstream main `bdc1eb02` (V: `git merge-base`).
- Not run: the determinism scenario hashes on lab builds of the base and the head (expected to change only after a multi-monster Fire Ball or Glacial Spike burst, at the footprint edge of a standing missile, for a melee blow's fraction, against an undead, after a Glacial Spike freeze, and after a Smite against a blocking monster); MSVC (read for shadowing, `min` / `max` / `near` / `far` identifiers and unchecked narrowing only: no new one, every `std::max` is qualified); a live game.
- The repair, melee and Smite harnesses are not part of this PR (they live outside the repo tree and can be attached): repair `vpr_oracle.py` with `vpr_probe.inc` and `cr_repair_game.py`; melee `sk_melee2.py` with `sk_melee2_test.cpp` and an independent `vs3_game.py` with `vs3_d2d.cpp`; Smite `vs_oracle.py` with `sbp_probe.cpp`.

## Answers to the presubmit warnings

- siblings (merc, pet, partner): components 2 and 3 apply to every caller of `player_blow`, so a merc's blow and a Shadow's blow take the same rules; a pet builds a simple fighter without Blunt or stat 122; components 4 and 5 touch monsters only; component 6 applies to a player's Smite, and the `Blow` of pets, missiles and deaths never has `swing.smite`, so for them `blocked` still means not hit.
- doc-claims `store.md`: "all" and "never" are true for vanilla data and the text says so.

## How to re-run

`test_freeze_rules --freeze-rules cases.tsv` against `freeze_rules.py gen|game|compare`; `test_sk_ground`, `test_sk_ground --hit ROW FRAMES PER` and `--burst-roll NAME TRIALS` against `sk_ground_hit.py` and `sk_ground_roll.py`; the Smite tests are in `test_combat`.

Agent: Claude Code
