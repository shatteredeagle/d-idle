# Review: revive refill candidate r1 (TASK-125), independent Opus verification

> status: verification of fork branch `local/fid-revive-refill-r1` (c3dac20a, one commit on upstream main 9c78f391), 2026-11-07, fidverify2 (Opus). Re-derived from the binary with an own harness and own d2d builds; the previous agent's conclusions were not used as input. A new private branch `local/fid-revive-refill-r2` moves the test into its own file (section 6); the code change is unchanged. Evidence it checks: [fid-revive-refill.md](fid-revive-refill.md), [pr-revive-refill-draft.md](pr-revive-refill-draft.md), the verifier brief in [pr-bundle-queue.md](pr-bundle-queue.md).
>
> Tags: **V read** read in game.exe 1.14d disassembly (upstream `tools/emu/disasm.py`, an own `--all` sweep and scans); **V run** game.exe code run in the upstream emulator by this review's own harness, or d2d tests built by this review; **I** inferred; **U** unverified; **S** opinion.

## Verdicts

| # | Claim | Verdict |
|---|---|---|
| C1 | FUN_0054c0e0 sets stats 6, 8, 0xa to the maxima of 7, 9, 0xb (FUN_00625d10 / 25d60 / 25db0) at 0x54c17c / 0x54c18a / 0x54c198, no compare or clamp | **HOLDS** (V read; V run on 610 own cases: 610 / 610 match, with and without the real stat callback) |
| C2 | only while the unit is dead (mode 0x11); nothing else calls it | **HOLDS** (V read: the dispatcher's arm and the handler's own check; the only pointer to the handler in the image is the table word; V run: a living unit changes in 0 / 610 cases, and 0 / 508 in r1's oracle) |
| C3 | d2d's `Fight::revive` refilled life only; the commit makes d2d agree | **HOLDS** (V run, own builds: own cases parent 597 / 610 differ, r1 0 / 610; r1's oracle on a new seed parent 489 / 508, r1 0 / 508) |
| C4 | the maxima include item and state bonuses (full stat-list getters), and d2d's maximum is the same | **HOLDS WITH LIMITS**: game.exe's target is the full list (flat and percent modifiers, V run, 373 cases with modifier lists). d2d's target is `stats.values[kMax*]`, which d2d's per-tick refresh keeps as own + items (+ Battle Orders). Whether d2d's sum equals game.exe's for real gear belongs to the item-stat subsystem, not to this change (F3, F4) |
| C5 | the death-time resets (curses, states, potions, buffs, gold, durability, corpse) are not claimed and not changed | **HOLDS** (the diff and the PR body read) |
| C6 | the commit and the PR body meet the maintainer's rules | **HOLDS WITH ONE CHANGE**: the test edits `tests/game/test_game.cpp`, which conflicts as text with open PR #242 and breaks the house rule "a new test is a new `test_*.cpp`" (F1). r2 fixes it |

Overall (S): the code change is right and the claim is proven. **READY to open as r2** (not r1). With #242 open, the cap of 3 open PRs allows it. Co-op ghost decision: keep the refill in the shared `Fight::revive` (F5).

## 1. The respawn path, read again (V read)

- Client packet table 0x6e0d18, 8 bytes an entry: entry 0x41 = (0x54c0e0, 1) (V run, read from the loaded image). The image holds the dword 0x0054c0e0 only in that table word (a byte scan of every section, section 6), and the linear sweep has no direct `call`, `jmp` or `push` of 0x54c0e0. So only the dispatcher reaches the handler.
- Dispatcher FUN_0054d750: the packet byte must be below 0x67 with a non-null table entry (0x54d772); then `eax = id - 0x14`, the byte table 0x54d848, the jump table 0x54d838 (0x54d790..0x54d797). For 0x41 the byte is 2 and the arm is 0x54d79e (V run, read from the image): `test esi, esi; je 0x54d821` (no unit: return 0), `cmp [esi+0x10], 0x11; jmp 0x54d7be; jne 0x54d821`. Mode 0x11 = 17 = DD. d2d has `kModeDD = 17` (gamedata.hpp:861), and that is the mode `Fight::player_modes` waits for before `World::respawn` may run (strike.cpp:995..997, tick.cpp:327). Other packets take the default arm (FUN_0057eec0).
- Handler FUN_0054c0e0 (0x54c0e0..0x54c293), in call order:
  - The packet length must be 1, and `cmp [esi+0x10], 0x11` checks the mode again at 0x54c107.
  - FUN_005531c0 (0x54c11e) returns the player data's object (`FUN_006221a0(unit)+0x9c`). FUN_00538670(obj, 4) tests its flag word. If the flag is set, the handler calls FUN_0052caf0(.., 3) and returns 0 without touching a stat (hardcore; V run below).
  - Otherwise FUN_0053fdf0 (0x54c14d; its body is `xor eax, eax; ret`, a constant 0 handed to the level change) and FUN_0056dfa0 (0x54c157, the item state re-apply).
  - **Then** the three maxima are read (0x54c15d, 0x54c165, 0x54c16e) and the three stats set (0x54c17c stat 6, 0x54c18a stat 8, 0x54c198 stat 0xa, layer 0). There is no compare, no `min` and no branch between the read and the write.
  - After the sets: FUN_00548520 three times (the client update, with the same register values), `or [esi+0xc4], 2`, state 0x36 on then off (FUN_00639db0), the town lookup (FUN_006427f0, FUN_00620bb0, FUN_0061a1b0, FUN_0061ab70) and the level change FUN_0053aec0 (0x54c216).
  - Then FUN_005809d0 (0x54c229). Its arguments are mode 1 (NU) and a last argument of 1, which skips its FUN_0057eec0 check; it calls the mode table entry 0x6e1740 + 8 = FUN_0057f020.
  - Last, the two hand slots (FUN_00620190 / FUN_006201d0 -> FUN_005701b0).
- FUN_00625d10: a null unit or list gives 0; otherwise it sets `edi = 0x70000` and calls FUN_00624f60 with row 7 of ItemStatCost. FUN_00624f60 reads the **full** array (+0x48) when the list's flag 0x80000000 is set, and the base array (+0x24) otherwise. It then has a minimum: on a full list whose owner (+0x44) is a player or a monster, and whose ItemStatCost row has the flag `[row+5] & [0x6ce270]` (= 4), a value below `[row+0x2c]` becomes `[row+0x2c] << [row+0x18]`. Rows (V run): stat 7 has the flag, MinAccr 1, shift 8; stats 9 and 0xb have the flag with MinAccr 0. A missing stat returns 0 before that minimum (0x624f81), so the minimum only acts on a stored maximum below 1, that is a negative one. r1's oracle builds its list without an owner (+0x44 = 0), so this branch cannot run there. This review builds the list with the real FUN_00626d40, which sets the owner; there is no difference for any non-negative maximum (F6).
- **No other writer after the sets** (V read, with limits). Method: a scan of every call to FUN_00627260 / FUN_006272b0 / FUN_006270b0 whose stat argument is 6, 8 or 0xa (138 sites), checked against the direct-call closure (depth 8) of each function the handler calls after 0x54c198. Hits:
  - in FUN_0053aec0's closure: FUN_005424f0 (0x54284a) and FUN_00573cb0 (0x573fc8). Both set stats 7 and 6 of a monster being spawned (`cmp [esi], 1` at 0x54284f).
  - in FUN_005701b0's closure: FUN_0056bfe0 (0x56c0f9) and FUN_005d2b60 (0x5d2b8d / 0x5d2c13), a skill's mana cost and the Blood Mana life cost (state 0x72).

  None of them writes to the respawned player (I). Indirect calls (several hundred in those closures) are not followed (U). FUN_005809d0's mode function FUN_0057f020 has no writer in its closure.
- Before the respawn (context, not on the claim's path): the death start sets life to 0 (0x57f337) and next calls FUN_00639fb0(unit, 0) (0x57f345; I: it clears the unit's states). The corpse takes every worn item and the item in hand (upstream `combat.md`, the "Dying" lines, FUN_0057fca0 -> FUN_0057f700). So at the respawn the full maxima no longer carry worn gear; inventory items still count (I).

### Other paths that refill (C1, checklist item 1)

| Path | game.exe | Reaches FUN_0054c0e0? | d2d | Tag |
|---|---|---|---|---|
| respawn after death, softcore (C->S 0x41) | 6, 8, 0xa := full maxima | yes | `World::respawn` -> `Fight::revive` (world/setup.cpp:458) | V read, V run |
| the same, hardcore | FUN_0052caf0(.., 3), return 0, no stat set | yes, then returns | no hardcore respawn in d2d's single player | V read, V run (610 / 610 unchanged, FUN_0052caf0 called once each) |
| a player's Resurrect skill, a shrine or an item | none found: no other writer of 6, 8 or 0xa that is a revive, and no other pointer to the handler | no | d2d co-op's Resurrect channel (`World::raise_step` -> `World::revive` -> `Fight::revive`) is d2d's own feature (deviations.md, local co-op row) | V read (scan); I that no such game.exe path exists |
| a co-op ghost raised, or revived in a town | none (no counterpart) | no | `World::revive` (world/setup.cpp:87, called from line 120 and tick.cpp:564) | S, F5 |
| a merc resurrected at a hire NPC | FUN_00579c00 sets the merc's stat 6 only (0x579ce6) | no | `cmd::ResurrectMerc` -> `rules::resurrect_merc`, not `Fight::revive` | V read; not changed by r1 |
| level-up | FUN_00570880 sets 6, 8 and 0xa (0x57094f, 0x570981, 0x5709b3) | no | its own refill (combat.hpp) | V read; out of scope |

## 2. Own harness and runs (V run)

`fv_revive.py` lives in this review's scratch clone, not in the PR; it imports only upstream `skill_tips.boot`. It differs from r1's oracle on purpose:
- The unit's stat list is built by the real FUN_00626d40, which sets the owner (+0x44) and the game (+0x60). The real stat callback FUN_0055b800 can be registered as well.
- Gear-like modifier lists are made by the real FUN_006251f0 (owner type 4), filled by the real FUN_006270b0 and attached by the real FUN_00626e10(unit, list, 1).
- FUN_00538670, FUN_0053fdf0 and FUN_0056dfa0 run for real.
- Hooked: FUN_005531c0 (returns a buffer whose flag word the case sets), FUN_00548520 and FUN_00639db0 (both recorded), the level change, the mode set, the hand slots, the town lookup, and FUN_0052caf0 (recorded).

Own generator, seed 20261111: 10 edge cases and 600 random ones.
- Edge cases: no gear; flat +40 life / +25 mana / +10 stamina, in one list and in two; +20 % / +15 % (stats 76 / 77); flat plus percent; negative modifiers; fractions; maxima 0; maxima of 100/256; currents above the maximum.
- Random cases: 70 % dead at life 0, with 0 to 3 lists. 373 of the 610 cases carry a modifier list; in 372 of them the full maximum differs from the base one.

| Run | Result |
|---|---|
| `game` (mode 0x11) | life, mana, stamina == FUN_00625d10 / 25d60 / 25db0 read before the handler: **610 / 610**; the maxima unchanged by the handler in 610 / 610; FUN_00548520 sends 6, 8 and 0xa with those values; state 0x36 on, then off |
| `game cb` (the real FUN_0055b800 registered) | 610 / 610 (the callback rescales current life while the lists attach; the handler's sets of 6, 8 and 0xa fire no arm) |
| `alive` (mode 1) | stats unchanged in 610 / 610 |
| `hc` (flag word 4) | stats unchanged in 610 / 610; FUN_0052caf0 called once each; nothing sent |
| examples | flat: 91.0 + 40 -> 131.0 (33536); +20 %: 23296 * 1.2 = 27955 (truncated); flat + %: (23296 + 10240) * 1.2 = 40243; mana (11520 + 6400) * 1.15 = 20608; -30 life: 15616; fraction 23373 + 773 = 24146, kept |
| d2d `World::respawn` (r1's `test_game --revive-stats`, own build) on the same cases, maxima = game.exe's full maxima | r1: **0 / 610** differ; parent (`fight/setup.cpp` of 9c78f391): **597 / 610** |
| r1's own oracle `fr_revive.py`, NEW seed 31415926, 508 cases | r1 **0 / 508**; parent **489 / 508**; `alive` changes 0 / 508 |
| `test_game` (full) | r1 rc 0, 29 `OK` lines incl. "OK: a respawn refills life, mana and stamina"; parent rc 134 at test_game.cpp:1304 (`got[0] == 90 * 256 + 77 && ...`) |

The percent rule game.exe applies (a percent of own + flat, truncated) has the shape of d2d's refresh (`raw = maximum - item_max - bo + max_flat * 256; item_share = raw * max_percent / 100`, components/game/fight/strike.cpp:172..193). That the two are equal is I (read, not run through d2d with real items).

Commands (worker host, nice 19, -j2, own build dirs; `fv/` is the scratch dir and never goes in the PR):
```sh
uv run python fv_revive.py info | gen 20261111 600 | game CASES [cb] | alive CASES | hc CASES
python3 check.py count game.tsv; python3 check.py d2dcases game.tsv > d2d_cases.tsv
build-fixed/tests/test_game --revive-stats d2d_cases.tsv | grep -E '<6 numbers>' > d2d.tsv; diff gamevec.tsv d2d.tsv | grep -c '^<'
uv run python fr_revive.py gen 500 31415926 > fr_cases.tsv; ... game / alive; test_game --revive-stats fr_cases.tsv
```

## 3. d2d side (V read)

- r1 changes only the first statement of `Fight::revive` (fight/setup.cpp:120): life, mana and stamina := `stats.values[kMaxLife / kMaxMana / kMaxStamina]`. The clears after it are untouched (regen, the stamina potion, potion states, charges, self states, curse and mark slot 0, stun, mode, merc placement), and the comment in fight.hpp:486 gains "mana and stamina".
- `stats.values[kMax*]` is the full maximum: `Fight::update_fighters` runs every tick for every player (tick.cpp:257, before the death and respawn block at 327) and adds the items' flat and percent shares and Battle Orders' share to it (strike.cpp:172..193). `make_corpse` moves every equipped item to the corpse at DD (world/travel.cpp:225), so by the respawn the refresh has already dropped the gear, as game.exe's corpse does (I).
- Battle Orders: d2d keeps `self_states` through the death and clears them inside `Fight::revive`, so the refill target still holds Battle Orders' share; the next tick's refresh lowers the maximum and clamps current to it. game.exe clears states at the death start (I, 0x57f345). Net effect: for one tick d2d's life and mana exceed game.exe's (F4).
- PR #242 (load life/mana, head 2cf72cb1) touches `World::take_character` (world/setup.cpp), the load path; r1 touches `Fight::revive`, the respawn path. The code is disjoint and nothing is applied twice: a respawn never runs the loader, and a load never runs `Fight::revive`. They do conflict as text in `tests/game/test_game.cpp` (V run: a `git merge` of r1 and pr/242 on 9c78f391 stops on that file): F1.

## 4. Claim limits the PR body already states, checked

- The co-op ghost shares `Fight::revive`: stated (honest limits). The death-time resets (curses, states, potions, buffs, gold, durability, corpse) are not claimed: the body says so and the commit does not touch them (C5). Hardcore and the death penalty are not claimed: stated.
- "no clamp, fraction kept": true (V run: the fractions 77 / 99 / 13 are kept; currents above the maximum are lowered to it, which is a "set", not a "clamp").
- "maximum 0: all become 0": true for game.exe with a real owner too (a missing stat returns 0 before the minimum, section 1).

## 5. Findings, ranked

| # | Sev | Finding | Recommendation |
|---|---|---|---|
| F1 | medium | r1's test and `--revive-stats` live in `tests/game/test_game.cpp`. That is a textual merge conflict with open PR #242 (both add a helper, a mode and a check at the same three places). Upstream AGENTS.md section 7 also says a new test is a new `test_*.cpp` and a file should not grow past about 1,500 lines; test_game.cpp is 1,980 lines on 9c78f391, and oracle-backed tests such as `test_cast_mana.cpp` have their own file | Open r2: the same code change, the test in a new `tests/game/test_respawn_refill.cpp` (`// test-links: game`), `test_game.cpp` untouched, and `fr_revive.py`'s usage line pointing at the new test. The body's re-check commands: `build/tests/test_respawn_refill --revive-stats` |
| F2 | low | The PR body has a "Claims (a batch only)" table for a single claim; the template says to delete sections that do not apply | Drop the Claims section (keeping it is harmless) |
| F3 | low | C4's "d2d's maximum is game.exe's" rests on d2d's item-stat refresh, not on this change; the oracle's d2d side (r1's and this review's) feeds the maxima in | Keep the body's limit sentence and add: "d2d's maximum is its refreshed sum of own, items and Battle Orders; this PR does not change how it is computed" |
| F4 | low | With Battle Orders up at death, d2d refills to the maximum including its share and clamps one tick later; game.exe clears the states at the death start | Mention it under "Not claimed" (states), or leave it out. A fix that clears states at death is D2 of the trace and needs its own trace |
| F5 | low (S) | The co-op ghost raise also gets full mana and stamina. Decision: **keep the refill in `Fight::revive`**, for three reasons. `World::revive`'s comment already says "full life and mana" (world/setup.cpp:95). The design note says everyone dead "respawns in town, as in single player", so one "back on your feet" state for both is the least surprising. Splitting it would leave a raised ghost at its death-time mana with nothing to justify that. A maintainer who wants the co-op path untouched can move the loop into `World::respawn` in one line | Say it in "What changes" (one sentence), not only in the limits |
| F6 | low | r1's oracle builds the stat list by hand without the owner (+0x44), so FUN_00624f60's minimum branch (stat 7, MinAccr 1) cannot run in it. No effect for non-negative maxima (section 1) | Optional: build the list with FUN_00626d40, as this review did. Not a blocker |
| F7 | info | presubmit `--quick`'s `gcc` step FAILs on `apps/launcher/main.cpp` (`main.moc: No such file or directory`), and it fails the same way on the **parent** tree with the parent's build (V run, presubmit's own `check_gcc`). It is environmental (the build dir has no Qt moc output), not this change | Say "pre-existing, also on 9c78f391" in the checklist line |

## 6. Rules check and r2 (C6)

- r1 c3dac20a and r2: author and committer `shatteredeagle <26069301+shatteredeagle@users.noreply.github.com>`; a subject plus 3 body lines; last line `Agent: Claude Code`; no Co-Authored-By (V read, `git show -s`).
- r1: `pr_guard.py up/main HEAD` 0 FAIL, 0 WARN; `--sorted` 0 FAIL; GCC `-Wshadow -Wconversion -Wno-sign-conversion -Werror` (presubmit's `check_gcc`) ok on the two touched TUs; the launcher failure reproduced on the parent (F7).
- r2 = `local/fid-revive-refill-r2` (170d920e, one commit on 9c78f391, pushed to the private fork remote; r1 left as it is). Changes against r1: `tests/game/test_game.cpp` untouched; the same helper, mode and two-class check in a new `tests/game/test_respawn_refill.cpp` (`// test-links: game`, the skeleton of `test_cast_mana.cpp`); the usage line of `tools/emu/fr_revive.py` names `test_respawn_refill`. Same subject and body, same identity. 4 files: fight.hpp, fight/setup.cpp, test_respawn_refill.cpp, fr_revive.py.
- r2 runs (V run, own builds, fixed and parent `fight/setup.cpp`):

| Check | r2 | parent |
|---|---|---|
| `test_respawn_refill` (no argument) | rc 0, `test_respawn_refill: ok` | rc 134, assertion at test_respawn_refill.cpp:75 |
| `test_respawn_refill --revive-stats` on this review's 610 cases | 0 differ | 597 differ |
| the same on r1's oracle, seed 31415926, 508 cases | 0 differ | 489 differ |
| `test_game` (full) | rc 0, 28 `OK` | rc 0, 28 `OK` |
| `pr_guard.py up/main HEAD` / `--sorted` | 0 FAIL, 0 WARN / 0 FAIL | |
| presubmit `check_gcc` on the two touched TUs | ok | |
| clang-tidy on the two touched TUs | no finding in a touched file (findings only in untouched headers: drlg/outdoor.hpp:1068, rules/merc.hpp:104) | |
| presubmit `--quick --base up/main` | 1 FAIL, 3 WARN: FAIL `gcc` = the launcher `main.moc` only (F7, reproduced on the parent); WARN docs-sync (no `.md` changed: answered in the body), WARN build (skipped by `--quick`), and WARN doc-claims (a heuristic on existing docs; this branch adds no `.md`); clang-tidy ok on 311 TUs (no finding on an added line); short names, sorted, ruff, attribution and tool tests ok | |

Note for anyone re-running presubmit in a fork clone: without `--base up/main` it diffs against the fork's own `origin/main`, which is behind upstream, and reported 8 FAILs here, all in files this branch never touched. Pass the base.

## 7. Body changes for r2 (for the PR draft's owner)

- "What changes": add one sentence: "d2d's co-op ghost raise shares `Fight::revive` and also comes back with full mana and stamina (no game.exe counterpart)."
- "How to re-check": `../../build/tests/test_respawn_refill --revive-stats /tmp/cases.tsv | grep ...`; "`test_respawn_refill` fails on the parent commit (assertion at test_respawn_refill.cpp:75) and passes with the change".
- Drop "Claims (a batch only)" (F2). Checklist: the presubmit line says the launcher `gcc` failure is pre-existing on 9c78f391 (F7).
- Honest limits: add F3's sentence.

## Proposed glossary term (not written)

"Respawn (C->S 0x41)" versus d2d's `revive` (also the co-op ghost raise) and the merc's resurrect at a hire NPC, three different paths: the trace asked for the same term.

## Shared-file lines for the merger
See `backlog/merge-lines/agent-fidverify2.txt`.
