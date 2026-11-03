# PR draft D2: a player blow against an undead takes the Blunt bonus and stat 122 (SKMELEE3, part 2)

> status: DRAFT, 2026-11-02 (card prbuildskmelee3b). ON HOLD: all upstream PRs are on hold; nothing is posted, nothing is pushed to a public remote. Branch `local/skmelee3-d2-final` on the private fork remote, head 283c7bce, one commit on upstream main cb716f10; independent of D1 (pr-skmelee3-d1-draft.md). The first version excluded mercs and Shadows (`not_player`); an independent check against the disassembly showed that gate is wrong and it is gone. Tags: V = run, S = from an earlier doc, U = unverified.

## PR text (for upstream; no private names)

**Title:** A player blow against an undead adds 50 for a Blunt weapon plus stat 122

**Body**

`FUN_0057b7d0` runs `FUN_0057b660` when the attacker is a player unit, or `FUN_006259b0(attacker)` is 2 (that function returns the alignment stat 0xac for players and monsters, 2 for any other unit type above 1; `0057b805..0057b828`, V read). Against a defender whose MonStats lUndead or hUndead flag is set (`FUN_0063e990`, masks 0x08 and 0x10, V run on game.exe) it adds 50 for a Blunt weapon (`FUN_00629bb0`: the ItemTypes Equiv chain to `blun`, or the ItemsTxt type2) plus stat 122 (`item_undeaddamage_percent`, `FUN_00625500`) to the record's enhanced damage, only when the sum is over 0. The bonus joins the same -90 floor as the gear percent. d2d had no undead input in the blow.

`Target::undead` comes from the monster type; `Fighter::blunt` and `Fighter::undead_pct` from `make_fighter`. One commit: `combat.hpp`, `d2s_items.hpp` (the stat name), `ai.hpp` (the new `Target` member) and `tests/test_combat.cpp`.

Who gets it: every caller of `player_blow`. That is the player's Attack, and also a merc's blow and a Shadow's blow (they are not excluded: the game's gate is "player or alignment 2", and allies carrying alignment 2 is S, not read live here; no merc class wields a Blunt weapon, so the practical effect is stat 122 on merc or Shadow gear, rare), and the weapon share of a skill missile (`Fight::strike` calls `player_blow` with `srcdam`; a test covers it). Kicks and Smite build their damage elsewhere and are unchanged; other pets use `simple_fighter` (no Blunt, no stat) and are unchanged.

Counted differential against the original game (game.exe's own functions under an emulator; the monster-row byte test and the item-type test are hooked in generator A and run natively in generator B), landed blows compared, whole points of the physical before the resistance (this PR does not touch the fine value; D1 does):

| generator | undead-target weapon cases | base differs (whole points) | this head differs |
|---|---|---|---|
| A, seed 7 | 32,304 | 19,198 | 0 |
| A, seed 1 | 32,314 | 19,204 | 0 |
| B, seed 23, 60,000 cases, all rows | 60,000 | 24,439 (armed, gear -90 and up, not floor: 15,670) | 8,811: 2,833 the one-point floor (D1's), the rest barehanded or the -90 floor, armed and clean 0 |
| B, seed 5, same | 60,000 | 24,585 (15,714) | 8,911: 2,814 floor, armed and clean 0 |

Generator A includes all weapon cases (290,847 landed over the two seeds): 0 whole-point differences on this head; on the base 38,402, all against undead targets (the mean gap is +50 points with a Blunt weapon). The remaining differences on this head are D1's one-point floor and the unchanged barehanded and double -90 floor cases.

Not changed and still different: barehanded blows, the -90 floor applied twice with a skill, off-weapon enhanced damage (none touched here).

Checks (worker host, build -j2 niced): seven suites pass on the head (data dir given); `pr_guard` 0 FAIL 0 WARN; `presubmit --quick` 0 FAIL, 2 WARN (docs-sync: no doc names this rule; build skipped by --quick); g++ and clang++ with the CI flags plus `-Wshadow -Wrange-loop-construct -Wconversion`, syntax only, over test_combat, strike, missiles, pets, merc and ai: 0 diagnostics on added lines; clang-tidy (CI's step): `tidy-pr-select` picks 227 TUs, those six were run: 0 findings in the touched files; no windows.h macro names or shadowed locals in the diff (read).

Overlap with open PRs: 174 adds `Target::crush_divisor` and changes `ai.hpp` at the same `Target` initializer, so a merge conflicts in two places (V, trial merge of head 489ef83d). Resolution: in `ai.hpp` keep 174's `crush_divisor(game_data)` argument and append `type_info.undead`; in `combat.hpp` keep both members (`int crush_divisor = 4;` first, then `bool undead = false;`; order matters, the initializer is positional). The resolved tree builds and passes test_combat and test_rules (V). D1 and D2 together conflict only in `tests/test_combat.cpp` (both insert a block at the same place; keep both blocks).

Determinism scenarios: not run (U). Expected: only scenarios with a hit on an undead by a Blunt weapon or with stat 122 gear change; the standard scenarios use neither. Checklist as in D1.

## Limits (U)

- Whether a hired merc or a Shadow has alignment 2 in game.exe (S): the rule is read from the disassembly, the alignment value of those units is not read live. If they have 0, mercs and Shadows get no bonus and the three call sites would need `not_player` back.
- Stat 122 is summed over all gear (`make_fighter`'s `sum`); the game reads the unit's own stat with `FUN_00625500`. Whether a shield's stat 122 is merged into the unit (the ItemStatCost damage-related flag of row 122) was not read; an attempt to read the table from the emulator failed (the tables are not loaded in a bare emulator). A stat 122 on a shield or armor is rare.
- That the ItemTypes equivalence matrix equals the transitive Equiv chain is S (it is built at load; the builder was not read).
- Missile weapon share: the gate also admits a missile attacker (type above 1 returns 2), so it takes the bonus; the game side of that path was not counted.

## Evidence (worker host, V unless said)

- Branch `local/skmelee3-d2-r174` 10e16ad5, base 48908176 (the merge of #174; earlier `local/skmelee3-d2-final` 283c7bce on cb716f10; the #174 conflict in ai.hpp and combat.hpp is resolved in this branch; counts re-run: rebase174-20261102.md), noreply identity for author and committer, trailer `Agent: Claude Code` only.
- Generator A: `sk_melee2.py` (U cases: undead 0 / 1, Blunt, stat 122 -60..50) with the build's `test_melee`, d2d md5 head 29d551c4 (seed 7), 2db2590a (seed 1). Generator B: the independent verifier's harness, header compiled in place.
- Disassembly read: `0057b805..0057b828`, `006259b0..006259ee`, `0057b660` (called at `0057b874`, before `FUN_0057b420` at `0057b8a4`).

## Shared-file lines for the merger

See openspec/changes/prbuildskmelee3b/merge-lines.txt.
