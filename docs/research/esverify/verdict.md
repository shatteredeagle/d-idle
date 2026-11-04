# Independent verification: Energy Shield absorbs the raw field (fork branch local/es-absorb-order)

> status: verifier verdict, 2026-11-04 (job esverify, docs and harness only; no game, no Go change, nothing pushed). CLAIM under test: head `12b78b60` (one commit on upstream main `2d051d77`) makes d2d's Energy Shield equal game.exe 1.14d's for the cases the author's harness covers. Tags: **V run** (run here, command below), **V read** (disassembly or source read here), **S** (another doc), **U** (not checked).

## Verdict

**CONFIRMED WITH CAVEATS.** The game side is now game.exe's own `FUN_005ca6c0`, run natively inside the real `FUN_0057c6c0` / `FUN_0057c1e0` chain, not a formula in a hook. Against it the head is exact on mana everywhere and exact on life wherever the frame has no flat reduction or magic damage reduction and no field above 4,096 points. Caveats C1 to C4 below.

## Harness (own, not the author's scripts)

| Piece | File | What it does |
|---|---|---|
| game side | `tools/emu/esv_game.py` | per blow, the real `FUN_0057c6c0` (flag 1) on a record with +8 / +0x10 / +0x1c / +0x24 / +0x20 = whole rolls << 8; the dispatcher `FUN_005c0c30` is a hook that, at event 11 for the wearer, rewrites its return address to a trampoline which calls the REAL `FUN_005ca6c0(wearer, attacker, record, 58, level)` and returns to the dispatcher's caller, so the resistance pass then runs on what the shield left. Hooked, not the claim: `life_apply.Game`'s list (stat dict, unit kinds, side effects), `FUN_006256b0` (state 30 present until removed), `FUN_00646ca0` (calc1 / calc2: the value d2d's own calc gives at the level), the state removal calls (logged), a fake skills row at `[0x744304]` put in only for the call |
| d2d side | `tools/emu/esv_probe.cpp`, `esv_build.sh` | the real `rules::monster_blow` (`sure`, `exact`), `Foe::take`, `Fight::absorb` of each tree, built from clean worktrees of `2d051d77` and `12b78b60` (Release, the worker host) |
| cases | `tools/emu/esv_cases.py` | edge grid (seed 0) plus seeds 7, 8, 9 x 20,000 (the author used 1, 2, 3); up to 3 blows a frame, physical plus up to three elements a blow, resistance -100..100, physical reduction 0..50 %, flat reduction and magic damage reduction 0..15 (20 % of cases each), levels 1..99, mana 0..600 points with fractions and edges (0, 1/256, 255/256, 0x10000, 0x100001, 0x7fffff00). Every shield case has a control twin: the same frame with no shield |
| compare | `tools/emu/esv_cmp.py`, `esv_fail.py`, `esv_regress.py` | per class, life (whole points and 256ths) and mana (whole and 256ths); "ctrl-exact" = the control twin agrees in 256ths on game, base and head, so a difference there is the shield's |

Run: `tools/emu/run-on-worker.sh --ttl 2800 esv_run.sh SEED N` for `0 500`, `7 20000`, `8 20000`, `9 20000` (V run). Harness self-checks (V run, every run): event 11 fired with the record still holding the raw fields in every call (snapshot_not_raw 0); no hook was called with arguments outside the expected ones; calc tables equal on base and head; the head's `test_energy_shield` and `test_combat` return 0.

Native check of the first differing case of the author (`esv_smoke.py`, V run): fire 632, resistance -100, calc1 40, calc2 32, mana 103,179/256: game.exe 220,406/256 life (860 points), mana 1/256. Equal to the author's number and to the head.

## Counts (V run)

Shield cases whose control twin is exact (the shield's own effect), seeds 7 + 8 + 9:

| Check | Cases | Base differs | Head differs |
|---|---|---|---|
| life, whole points | 24,174 | 4,494 | 171, all with a flat reduction or mdr (C1) |
| life, 256ths | 24,174 | 6,600 | 209, all with a flat reduction or mdr |
| same, frames with no flat / mdr | 15,621 | (most of the 4,494) | **0** life, **0** mana |
| mana, 256ths (all shield cases, any control) | 51,065 | 4,428 | **0** |
| no-shield frames: base output == head output | 60,000 | | equal in 60,000 (no change off the shield path) |

Edge grid (seed 0, 8,802 shield cases, 5,500 ctrl-exact): base life 479, head 25 (24 are C2, one is C1); mana head 0 in all 8,802.

All shield cases (control not filtered), per seed: head life whole-point differs in about 5,100 of 17,000, but the no-shield frames differ in about 750 of 3,000 the same way on base and head: d2d resists each element in whole points while game.exe keeps 256ths and sums the record (no-shield multi-field frames 726 of 2,366, seed 7). That is not this change. Cases where the base matched game.exe and the head does not: 21 without flat / mdr, all in frames whose control already differs (coincidental agreement), and 123 with flat / mdr (C1).

## Break attempts (V run)

| Attempt | Result |
|---|---|
| mana 0, 1/256, 255/256, exactly at the cap, 0x7fffff00 | head equal on mana in every case; life equal |
| resistance 100 (immune), -100 | equal: game.exe pays mana for an immune field, the head does too |
| field 4,095 / 4,096 / 4,097 / 12,000 points (MulDiv's 0x100000 boundary) | mana equal; life differs on physical with a percent reduction (C2) |
| level 99 (calc1 above 95 before d2d's clamp) | d2d's calc returns 95 at the levels tried (`calc.txt`); no case reaches the clamp |
| mana running out inside a record, across records of one frame | equal (mana 0 differences) |
| flat reduction and mdr with a high-level shield | breaks: C1 |

## Caveats

- **C1 (medium, V run).** A field the shield leaves below the flat reduction (physical) or the magic damage reduction (elements) goes NEGATIVE in game.exe and is summed into the record total, so it takes damage off the other fields (field probe `esv_probe8515.py`: physical 215 less 95 % = 2,752/256, flat 15 points: game.exe keeps -1,088/256 and the record deals 925/256; head 2,013/256). d2d clamps each part at 0 (`resisted_fine(..) - mdr`, `reduce_fine`); the base clamped too (its comment on `reduce_fine` says so). The shield makes this common: 2.5 % of the clean flat / mdr frames (171 of 6,535 over three seeds), and flat reduction and mdr are ordinary gear. The PR's numbers had flat and mdr at 0, so its "0" does not say this; the body should name it as a limit or a follow-up.
- **C2 (low, V run).** A physical field above 0x100000 (4,096 points) with a percent reduction: game.exe divides first (`(fine / 100) x (100 - pct)`), `reduce_fine` multiplies in 64 bits. Pre-existing in `reduce_fine`; only shows here because the shield moves the field. Not reachable by real monster hits.
- **C3 (low, V read + V run).** `FUN_005ca6c0` removes the state whenever the mana it leaves is 0, also when it absorbed nothing (mana 0 on entry), but not when calc1 is 0 (it returns before the loop; V run: level with calc1 0 kept the state). Removed in 9,232 to 9,338 of about 17,000 shield cases per seed: in play the shield ends on the first hit at empty mana. The PR leaves this to a follow-up (it says so); the single-frame claim is not affected, later frames are.
- **C4 (low, V read).** The three monster-only fields (+0x38, +0x3c, +0x40) are absorbed only when the attacker is a monster that is not a hireling (`0x5ca79b..0x5ca7aa`); this harness's attacker was a player unit and the fields were 0, so that path is untested. d2d has no such fields (the PR says so).

## Out-of-scope limits the PR names (brief item a)

| Limit | Own deviation? | Makes this PR's claim misleading? |
|---|---|---|
| shield after the armors; game.exe runs the last registered absorb first | yes, separate (needs Bone or Cyclone Armor with Energy Shield: another class's skill, only by oskill items) | no, if the body keeps "with an armor up the order is not matched" |
| state not removed at 0 mana | yes, separate (sk-armor-trace D3); frequent in play (C3) | no for one frame; the body should say the effect across frames is larger than "unchanged here" suggests |
| 2,262 magic-with-resistance rows | yes, separate (`monster_blow` has no percent magic resistance) | no; the rows are reported, not counted. Excluding them is fair since d2d has no rule for them on either head |

## PR rules and text (brief items b to d)

- (b) V read: author and committer the numeric noreply identity, body 2 lines plus `Agent: Claude Code` (AGENTS.md "at most three lines of body"; the lines are 115 and 164 characters, long but allowed). `python3 tools/pr_guard.py 2d051d77 12b78b60` on the head worktree: 0 FAIL, 0 WARN (V run). Upstream main has moved to `561814df`; `git merge-tree` of the head on it is clean and none of the five touched files changed (V run).
- (c) V run: clang-tidy with the repo's `.clang-tidy` on `tests/game/test_energy_shield.cpp` and `components/game/fight/states.cpp`: no finding in the touched files; the only errors are `readability-identifier-length` in `components/drlg/outdoor.hpp`, not touched. The test uses `fs::` from the `game.hpp` alias it reaches through other headers; include-cleaner did not flag it here (V run, this clang-tidy).
- (d) Body text: "V run" for the differential is accurate for the order and the resistance step; the arithmetic is now V run here (this doc), so the body's "Not re-run: a real FUN_005ca6c0 run" can cite this verification instead. Overreach: "for every blow the shield ... the rest goes through the resistance" is true, but "the flat magic reduction and the physical reduction work on what is left" hides C1 (d2d clamps per part, game.exe does not). The 60,000-case run was on `083b6edd`, not the final head (the body says so); this run is on the final head `12b78b60`. References upstream cannot repeat: the harness is not in the PR (`stateev_diff_*` and `esv_*` live in this repo), so the counts are "cited, not proven" for the maintainer unless the harness is attached; the in-tree `test_energy_shield` is the repeatable part. The earlier doc `stateev-diff-20261104.md` gives the base at seed 1 case 9 as 1,063 points and 267/256 mana, the PR body 1,062 and 1/256: the earlier figure came from the probe with the stray physical roll (fixed in `52a7a63d` by `sure`, `exact`; V read, the fix is right).
- (e) The bot: no absorb or damage-intake model in `internal/combat` (V read: Energy Shield appears only as a buff name and state 30); no plan in `internal/plandata` buys it. Not a gap today; if a plan learns Energy Shield, the bot's mana and chicken logic must take the shield from the raw hit, bill mana in 256ths, and expect the state to end at 0 mana (C3).

## Recommended actions

1. Add C1 to the PR body's limits (flat reduction and mdr: game.exe lets a part go negative and sums the record; d2d clamps per part), or fix it in a follow-up with a record-total clamp.
2. Replace "Not re-run: a real FUN_005ca6c0 run" with a pointer to a native run; quote counts from the final head.
3. Follow-ups as separate PRs: the state ending at 0 mana (C3), magic percent resistance, the element 256ths in `monster_blow` (the no-shield multi-field rows above).

## Reproduce

Worktrees of the fork at `2d051d77` and `12b78b60` under the verifier's worktree directory, mirrored by labrun; `labrun <worker> -- bash tools/emu/esv_build.sh base|head`, then the runs above. Outputs: `$D2_SCRATCH/run-SEED/` on the worker host (not kept in the repo).

## Shared-file lines for the merger

- BACKLOG: esverify: Energy Shield head 12b78b60 confirmed with caveats against game.exe's native FUN_005ca6c0 (docs/research/esverify/verdict.md); before posting, add C1 (flat / mdr parts go negative in game.exe) to the PR body limits; follow-ups: state end at 0 mana, magic percent resistance, element 256ths in monster_blow.
- INDEX: research/esverify/verdict.md: independent verification of the Energy Shield absorb-order PR, native FUN_005ca6c0 vs d2d base and head (60,000 random frames + edge grid)
- CHANGELOG: docs: Energy Shield absorb-order PR verified against game.exe's own FUN_005ca6c0 under the emulator (docs/research/esverify/verdict.md, tools/emu/esv_*)
