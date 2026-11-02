# PR draft: a melee blow keeps its 256ths and takes the undead / Blunt bonus (SKMELEE3)

> status: DRAFT, 2026-11-02 (card prbuildskmelee3). ON HOLD: the user put all upstream PRs on hold; nothing is posted, nothing pushed to a public remote. Branch `local/skmelee3-v1` on the private fork remote, head 81b7a0a0 (3 commits on upstream main 3852c904). Tags: V = run, S = from an earlier doc, U = unverified. An independent Opus verification of the built branch is still owed (the earlier verifier, docs/research/verify-skmelee2.md, used Sonnet).

## PR text (for upstream; no private names)

**Title:** A melee blow keeps its 256ths through the resistance; against an undead a Blunt weapon and stat 122 add damage

**Body**

Two rules of the player's melee blow (`player_blow`) differ from the original game. Each is its own commit.

1. `FUN_0057b420` returns the physical in 256ths and `FUN_0057bf80` resists that fine value; the life stat takes it as is. `player_blow` shifted the roll to whole points (and raised it to 1) before the resistance. The part under a point is now kept in `Blow::frac`, as `missile_blow` already does, and the one-point floor is gone. RNG draws are unchanged.
2. `FUN_0057b7d0` calls `FUN_0057b660` for a player attacker: against a monster with lUndead or hUndead it adds 50 for a Blunt weapon (ItemTypes Equiv chain to `blun`) plus stat 122 (`item_undeaddamage_percent`) to the enhanced damage, only when the sum is over 0. `Target::undead` comes from the monster type; `Fighter::blunt` and `Fighter::undead_pct` from `make_fighter`. A merc and a shadow are not player units, so their fighters drop both terms (`not_player`).

Counted differential against the original game (game.exe 1.14d's own functions run under an emulator, the same cases on both sides; case generator and the two harness files are in the tools folder of the author's repo and can be attached). The cases carry random weapons, enhanced damage, flat damage, Strength and Dexterity bonuses, skill damage percent, undead and Blunt flags and stat 122 in -60..50. Two seeds, 100,000 cases per part:

| weapon cases (landed blows) | compared | base: differ in 256ths / whole points | head: differ in 256ths / whole points |
|---|---|---|---|
| plain blow, seed 7 | 80,557 | 80,250 / 0 | 0 / 0 |
| plain blow, seed 1 | 80,951 | 80,632 / 0 | 0 / 0 |
| undead grid with weapon (non-blunt and blunt, stat 122 below, at and above 0), seed 7 | 64,688 | 64,488 / 19,198 | 0 / 0 |
| same, seed 1 | 64,651 | 64,487 / 19,204 | 0 / 0 |

That is 290,847 weapon cases: 289,857 differ in 256ths on the base, 0 on the head. The mean gap on the base is +0.498 points a landed blow (D1), and +50 points with Blunt against an undead. Cases where the build rolls a miss are excluded on both sides. Sub-point blows (gear percent -90 and below, skill damage 0): 0 differences on the head.

Not changed and still different (counted on the head, same cases): barehanded blows (Strength as a damage percent and the 1 / 2 range: 14,355 and 14,107 of the plain part, 32,303 and 32,246 of the undead grid), the -90 floor applied twice with a skill (40 of 377 cases), off-weapon enhanced damage (795 of 952), to-hit with attack rating at or under 0 (44 and 36 of 100,240). None of these is touched here.

Limits, stated plainly: the undead test and the Blunt test are the game's own functions with two parts hooked in the oracle (the monster row byte test and the item-type test); the masks of the monster test (8 and 0x10: lUndead, hUndead) are read (V run), the Blunt chain is the ItemTypes data (D). Kicks, Smite and conversion paths share the fine-damage change (they use the same code) but were not part of the counted cases (U). Stat 122 on a worn shield is counted by d2d (the whole sum) while the game takes the off hand's stats out (U, edge). Pets and mirror shadows keep the old integer behaviour for their own fighters (whole-point damage, so the fine change is identical for them: V by reading, an integer roll through a resistance gives the same whole points).

Overlap with open PRs: 159 touches `ai.hpp` and `strike.cpp`, 160 and 163 `missiles.cpp` and `fight.hpp`. A trial merge of this branch with each of the three is clean (V, `git merge-tree`).

Checks: the seven suites (test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game with the data) pass on the head; `pr_guard` 0 FAIL 0 WARN per commit; `presubmit --quick` 0 FAIL, 4 WARN answered above (siblings: merc and pet handled by `not_player`; melee only, missile skills share `player_blow` for their weapon share; no doc to sync); g++ 16 and clang++ 23 with the CI flags plus -Wshadow -Wrange-loop-construct -Wconversion: 0 diagnostics on added lines in pets, merc, strike, missiles and test_combat.

Determinism scenarios: not run for this draft (U). Expected effect: every scenario with a player or merc melee blow changes its monster-state hash by the whole-point carry (the part under a point now reaches life); scenarios without a melee hit (passive, cold, static, cold2) should not change. Checklist below.

## Evidence on the worker host (V)

- Branch `local/skmelee3-v1` (private remote), commits a9af1f69 (fine physical), 9d54beff (undead / Blunt), 81b7a0a0 (test shadow fix; squash before any PR). Base 3852c904.
- Differential: `tools/emu/run-on-worker.sh --ttl 2900 sm3_run.sh 100000 7 1700` and the same with seed 1; base and head test binaries from the same harness source (`tools/emu/sk_melee2_test.cpp` as `tests/test_melee.cpp`). Case file md5 (first 8) fa074d88 (seed 7), 265e56ee (seed 1); game side 4be2122f, f6ce6acf; d2d side base eba407be / 1809103f, head dca122a5 / 7256a969. The base run reproduces the SKMELEE2 counts (80,250 and 80,632).
- Masks: `tools/emu/run-on-worker.sh sm3_probe.py` read 0x6ce274 = 8 and 0x6ce278 = 0x10.
- Unit tests: tests/test_combat.cpp, two blocks (fine physical through resistances 0, 33, 50, 100, -50, a sliver under a point, skill flat; undead bonus: Blunt, stat 122 above, at and below 0, with skill ED, not_player, kick unchanged, make_fighter's Blunt chain by type and type2).

## Det scenario checklist (not run)

Build two lab trees (upstream main plus the lab patch set, `dethash2/build.sh`): base and base plus these commits; then `D2D_DATA=<data> d2dslot.sh run -- python3 det_scenarios_run.py run bin OUT passive,fight,cold,static,cold2,hit_plain,hit_champion,hit_unique,hit_boss,hit_immune,hit_minion,merc,pet,follow base,base2,head`. Read `compare`: base2 must equal base; list the scenarios whose hashes differ and attribute each to a whole-point carry. Needs a d2d slot and two full builds; not done here because of host load.

## U list

Opus verification of the branch; det hashes; kick / Smite / conversion under the fine change; shield stat 122; Shadow Master's real undead behaviour (read as not a player unit).
