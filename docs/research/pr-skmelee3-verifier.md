# Verifier: SKMELEE3 PR candidate (local/skmelee3-v1)

> status: current (2026-11-02, card verifyskmelee3, Opus verifier). Candidate head 81b7a0a0 on upstream main 3852c904 (private fork remote). Own harness first, the author's draft (docs/research/pr-skmelee3-draft.md on agent/prbuildskmelee3, c6724124) read afterwards. 75-minute limit: parts marked NOT RUN. Tags: V read (disassembly), V run (game.exe function under the emulator, or d2d's real header code compiled and run), S (inference / earlier doc), U (unverified). No game, no slot; nothing pushed or posted.

## Verdict (provisional): READY WITH CHANGES, split D1 / D2

- **D1** (fine physical, `Blow::frac`, no one-point floor): READY as far as checked. Own oracle agrees with head on every armed case outside the two known untouched deviations (below). Det hashes and the full suite run are NOT RUN, so it stays provisional.
- **D2** (undead / Blunt bonus): the bonus rule and both masks hold. But the `not_player()` exclusion of the merc and the Shadow is **not** what the disassembly says (finding 1). Fix that, or back it with a live alignment read, before posting. Posting D2 as its own PR after D1 is the cleaner split: D1 is a pure fidelity fix with no open question.

## Findings (ranked)

1. **HIGH: the merc / Shadow exclusion is not the game's gate (V read).** `FUN_0057b7d0` (0057b805..0057b828) runs the stat 120 / demon / `FUN_0057b660` / `FUN_0057b6a0` block when the attacker is type 0 (player) **or `FUN_006259b0(attacker) == 2`**, and the defender is type 1. `FUN_006259b0` (006259b0..006259ee) returns 2 for any unit type above 1, and for players and monsters it returns stat 0xac (alignment) from the state-0x69 stat list. So the gate is "player, or an attacker of alignment 2", not "player only". A merc or a Shadow that the game gives alignment 2 gets the Blunt and stat 122 bonus. That allies carry alignment 2 is S: palaura-trace.md reads "player 2, ordinary monster 0", and d2d itself models `align == 2` for the player's side. The candidate's comments in merc.cpp ("FUN_0057b660 is for a player unit only") and combat.hpp (`not_player`) state the wrong rule. Recommendation: drop `not_player` from merc.cpp and pets.cpp (the three call sites), so mercs and Shadows take the bonus as the player does. Or keep it only with a live read of stat 0xac on a hired merc and a Shadow showing it is not 2. Either way, cite `FUN_006259b0` in the comment. Player-visible size is small (no merc class wields Blunt; stat 122 on merc or assassin gear is rare), but the PR text states a rule that is false.
2. **MEDIUM: missile weapon-share path changes and was not counted (V read of d2d; game side U).** `missiles.cpp:681/693` builds the blow with `actor->player_combat` (no `not_player`) for any missile with SrcDam > 0. On head, a player's bow or throw skill against an undead now takes stat 122 (`undead_pct` is the whole-gear sum). It also keeps the fine physical, and `missile_blow` then adds its own frac on top, so frac can pass 255. `hurt()` carries any frac (ai.cpp:3047), so that is safe. Under the gate in finding 1 a missile-type attacker also qualifies, so this is probably right. But the PR claims "melee", and none of the counted cases cover it. Recommendation: say it in the PR text, and add one test for a SrcDam missile against an undead.
3. **MEDIUM: stat 122 is read from the whole-gear sum (S).** `make_fighter` sets `undead_pct = stat(kUndeadDamagePercent)` over all gear. The game reads it with `FUN_00625500` on the attacker (V read), and the author says a worn shield's 122 is not counted by the game (my S: a damage-related stat lives on the weapon only). Not run by me. Recommendation: use the weapon's sum (`weapon_sum`) if that is confirmed, or list it under the PR's limits.
4. **LOW: the comment on `Blow::frac`** (combat.hpp:385) still says "a missile's element", but player_blow now sets it too. Recommendation: reword it.
5. **LOW: untouched deviations stay as before** (they are not regressions): barehanded Strength term (D4) and the -90 floor applied twice (D6). My counts are below.

## Own harness (V run)

Files (scripts only) are in `docs/research/pr-skmelee3-verifier/`. On the worker host they run from my own scratch directory, not from tools/emu. Nothing in them comes from sm3_* or vm2_*, except the emulator library itself (upstream `tools/emu/emu.py`, `disasm.py`).
- `vs3_game.py`: calls game.exe's `FUN_0057b660` and then `FUN_0057b420`, with the record ED from the first as the ED of the second, as `FUN_0057b7d0` passes it. The undead test `FUN_0063e990` and the Blunt test `FUN_00629bb0` run **natively**, on fake MonStats rows (flags byte +0xd = 0, 0x08, 0x10, 0x18) and a fake ItemTypes equivalence bit matrix (type rows 10 and 13 blunt, 11 and 12 not; one case kind has the blunt type only in ItemsTxt +0x120 type2). Hooked: the stat getters, the weapon in hand, the item type and record lookups, StrBonus / DexBonus, mastery 0.
- `vs3_d2d.cpp`: d2d's real `rules::player_blow`, compiled twice against the base and head headers (`-DVS3_HEAD`). The Fighter is filled as `make_fighter` fills it. Resistance runs at 0, 25, -50 and 75; the game side is resisted with the `resisted_fine` rule (I for res != 0).
- Cases: seeds **23** and **5** (author: 7 and 1), 50,000 each. 85 % armed; gear ED 0..300; stat 25 down to -150; Str / Dex 10..249; skill ED -30..150; SrcDam 0x10..0xC0; 60 % undead rows; stat 122 from -200 to 100.

| seed 23 / seed 5 (res 0) | count |
|---|---|
| landed blows | 50,000 / 50,000 |
| base differs from game in 256ths | 49,875 / 49,893 |
| base differs in whole points | 20,397 / 20,485 (undead rows: 17,532 / 17,712) |
| base floor lifted a sub-point blow to 1 | 2,403 / 2,348 |
| **head differs, armed, gear p >= -90** | **0 / 0** |
| head differs, gear p < -90 (D6) | 2,749 / 2,777 |
| head differs, barehanded (D4) | 7,402 / 7,425 |

Res 25, -50 and 75 give the same head result (armed clean 0 / 0 on both seeds).

Break attempts: SrcDam 0x100 made the game return 0. SrcDam is a byte (`FUN_0057b7d0` reads `byte [ebp+0x14]`), so it is not a valid input and was dropped. Also tried: stat 122 at -200 against a Blunt weapon (sum under 0: no bonus, as head), type2-only Blunt (head equal), sub-point blows at SrcDam 0x10 (head equal, base floors to 1).

## Rules and addresses (V read)

- `FUN_0057b660`: `FUN_0063e990(target)`. Then, if the weapon (`FUN_00535bc0`) is ItemTypes 0x39 (57, blun) by `FUN_00629bb0`, +50, plus stat 0x7a (122) by `FUN_00625500`. Added to record +0xc only when the sum is over 0. Called at 0057b874, before `FUN_0057b420` at 0057b8a4, which takes record +0xc as its ED. The bonus therefore joins the same -90 floor as gear p.
- `FUN_0063e990`: unit type 1, class id under MonStats count (+0xa80), row 0x1a8, `byte +0xd` against [0x6ce274] = **0x8** or [0x6ce278] = **0x10** (read from game.exe: V run).
- `FUN_00629bb0`: the item type's row in the equivalence bit matrix (+0xbfc / +0xc00 / +0xc04); else the ItemsTxt +0x120 (type2) row. d2d's `type_is(type) || type_is(type2)` has the same shape. That the matrix is the transitive Equiv chain is S (it is built at load; the builder was not read).

## Readers of the change in d2d (V read)

Every `player_blow` result reaches `Fight::land` and then `hurt()`, which carries frac (strike.cpp:511; frac is dropped only when soak takes part of the hit, an existing ponytail). Effects of the change:
- `blow.phys` is no longer floored. It feeds the Weaken share (deaths.cpp:315) and the soak share.
- Get-hit now sees the fine size.
- Kicks (strike.cpp:818, `pf_kick`, no bonus branch) and Smite (frac kept, no bonus) take the fine change. Their game side is U.
- Conversion splits before the fine value, as before.
- Thorns (`player_thorns`, merc thorns) do not read `Blow`.
- Pets other than the Shadow use `simple_fighter` (blunt false). The traps in tick.cpp:413 use `simple_fighter`.

## Det scenarios: NOT RUN

The lab trees are built: base 3852c904 plus the 8 lab-overlay commits, and the same plus the 3 candidate commits, clean cherry-pick. The -j1 builds did not finish inside the limit. Expected (S): every scenario with a player, merc or Shadow melee hit changes its hash; passive, cold, static and cold2 should not.

## CI parity (partial)

- Clean merge with upstream main cb716f10: `git merge-tree` exit 0 (V).
- MSVC by reading: no windows.h macro names, no shadowed locals in the src diff, every narrowing is an explicit `int(...)` (V read).
- g++ 16 syntax-only on merc.cpp and pets.cpp with the CI flags plus -Wshadow -Wrange-loop-construct -Wconversion: 0 warnings (V). clang++: warnings only in unchanged headers (cube.hpp, ds1.hpp), sign-conversion. test_combat.cpp: g++ gives 37 set-but-unused warnings on head, 4 of them on the new lines (land_blow, mace, skeleton, foot). They are the same kind as the old ones and come from the -DNDEBUG I took from the Release command (asserts compiled out), so they are not a CI finding (S).
- Includes: merc.cpp, pets.cpp and test_combat.cpp include `combat.hpp` directly; combat.hpp includes `rules.hpp` (`type_is`) and `d2s_items.hpp` (V read).
- NOT RUN: clang-tidy, the 7 suites (the head test build was at 244/270), pr_guard, presubmit --quick.

## Author's claims vs my runs

- Agree: head 0 on armed blows outside D4 and D6; the masks are 8 and 0x10; untouched D4 and D6.
- Overclaim: "a merc and a shadow are not player units, so no bonus". The gate also admits alignment 2 (finding 1).
- Under-stated: the missile SrcDam path also changes (finding 2).
- The author hooked the monster-row and item-type tests; I ran both natively.

## Shared-file lines for the merger

See openspec/changes/verifyskmelee3/merge-lines.txt.
