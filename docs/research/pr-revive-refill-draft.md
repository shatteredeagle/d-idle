# PR draft: a respawn after death refills life, mana and stamina

> status: DRAFT, NOT POSTED, NOT VERIFIED. Candidate branch `local/fid-revive-refill-r1` (one commit `c3dac20a` on upstream main `9c78f391`, pushed to the private remote only). NEEDS independent Opus verification before opening (brief: docs/research/pr-bundle-queue.md, section "fid-revive-refill: verifier brief"). Proof and limits: [fid-revive-refill.md](fid-revive-refill.md). Own subsystem change; NOT part of PR #242 (character load). The body below follows the upstream `.github/pull_request_template.md`.

````
## What changes
When a dead player respawns, d2d set life to the maximum and left mana and stamina as they were at death. game.exe sets all three to their maximum. `Fight::revive` now does the same.

## Evidence from game.exe 1.14d
| Address | What it shows |
|---|---|
| `0x6e0f20` (table at `0x6e0d18`, 8 bytes an entry) | entry 0x41 of the client packet table is `FUN_0054c0e0` (already in `network.md` and `net-packets.md`) |
| `FUN_0054d750` `0x54d79e`..`0x54d7be` | the dispatcher runs entry 0x41 only while the unit's mode `[unit+0x10]` is 0x11 (dead) |
| `FUN_0054c0e0` `0x54c15c`..`0x54c179` | the three maxima are read: `FUN_00625d10` (stat 7), `FUN_00625d60` (stat 9), `FUN_00625db0` (stat 0xb), each the full stat list getter `FUN_00624f60` |
| `FUN_0054c0e0` `0x54c17c`, `0x54c18a`, `0x54c198` | stats 6, 8 and 0xa are set to those values with `FUN_00627260`; no comparison with anything |
| `FUN_0054c0e0` `0x54c19d`..`0x54c1c8` | the new values are sent to the client |

Oracle `tools/emu/fr_revive.py` runs the real `FUN_0054c0e0` (with the real maxima getters and stat setter; the helpers around it are hooked) on 408 dead-unit cases (life, mana and stamina below, at and above the maximum; maximum 0; fractions) and `test_game --revive-stats` runs `World::respawn` on the same values; `diff` of the two vectors: d2d before this change 390 of 408 differ, with it 0. A second seed: 0 of 1508 differ with the change. A control run of the same handler on a unit that is not dead changes nothing in 408 of 408.

## Claims (a batch only)
| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| this one | a respawn sets life, mana and stamina to their maxima, exactly (no clamp, fraction kept) | the addresses above; oracle 408 cases, 0 differ | `test_game --revive-stats` against `uv run python tools/emu/fr_revive.py game`; `test_game` prints "OK: a respawn refills life, mana and stamina" |

## How to re-check
```sh
python3 tools/pr_guard.py
cd tools/emu && uv sync
uv run python fr_revive.py gen 400 > /tmp/cases.tsv
uv run python fr_revive.py game /tmp/cases.tsv > /tmp/game.tsv
../../build/tests/test_game --revive-stats /tmp/cases.tsv | grep -E '^[A-Za-z_]\w* -?[0-9]+( -?[0-9]+){5}$' > /tmp/d2d.tsv
diff /tmp/game.tsv /tmp/d2d.tsv && echo same
```
`test_game` fails on the parent commit (`Assertion got[0] == 90 * 256 + 77 && got[1] == 40 * 256 + 99 && got[2] == 80 * 256 + 13 failed`) and passes with the change; the test runs two classes (Sorceress and Barbarian).

## Checklist
- [x] A test in `tests/` and the oracle in `tools/emu/`, in the same commit
- [ ] The finding in `docs/research/re/` (`combat.md` already says "life, mana and stamina full" for the resurrect, line 494; this PR adds the addresses of the three sets if the maintainer wants a row)
- [x] `python3 tools/pr_guard.py` passes (0 FAIL, 0 WARN)
- [x] `presubmit.py --quick` on the worker host (GCC Release build of `test_game`, clang-tidy on 310 TUs): ok for attribution, whitespace, tool tests, ruff, short names, sorted, clang-tidy (no finding on a line this branch added); WARN docs-sync (answered by the Checklist row above and this body), WARN build (skipped by --quick), dirty-tree (scratch helper scripts outside the tree). The `gcc` step FAILED only on `apps/launcher/main.cpp` (`main.moc` not generated: the build dir was configured for `test_game` only), a file this change does not touch. Replaced by a direct `-fsyntax-only -Wshadow -Werror -Wall -Wextra` compile of the two touched C++ files: rc 0 each

Honest limits: the oracle's unit has no gear and no state, so the maxima are the cases' own; that a worn item's or a state's bonus enters the maximum is read in the getter (`FUN_00624f60` is the full-list getter), not run in this oracle, and d2d's maxima already are the sum of the items. The helpers around the three sets (the move to town, the mode, the item states re-applied by `FUN_0056dfa0`, the client update, the hand slots) are hooked, so the oracle proves the stat vector only. `Fight::revive` is also called for d2d's co-op ghost raise (`World::revive`), which game.exe has no counterpart for; the ghost now comes back with full mana and stamina as well (the comment on `World::revive` already says "full life and mana"). Not claimed: the states, curses and potions `Fight::revive` clears (game.exe clears them elsewhere, not decoded), the hardcore branch of the handler, the death penalty and the corpse. No live read of a respawn from hurt mana on game.exe exists (the repo's earlier live drill read mana 40/40 after a respawn, not what it was at death).

---

Agent: Claude Code
````
