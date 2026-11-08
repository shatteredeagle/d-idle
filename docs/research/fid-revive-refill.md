# fid-revive-refill: what game.exe's respawn refills, and d2d's Fight::revive (TASK-125)

> status: DONE as a trace, candidate READY for independent Opus verification (not yet verified). Agent fidrevive, 2026-11-07. Candidate: local branch `local/fid-revive-refill-r1` (one commit `c3dac20a` on upstream main `9c78f391`, pushed to the private fork remote the same day, never to GitHub). PR draft: [pr-revive-refill-draft.md](pr-revive-refill-draft.md). Family of PR #242 ([fid-loadhp-trace.md](fid-loadhp-trace.md)) but a SEPARATE subsystem change (Fight::revive / resurrection), its own candidate; nothing here goes into #242.

Tags: V read (read in the disassembly), V run (game.exe's real code under the emulator, compared with d2d), S (shown by an existing upstream or repo doc), I inferred, U unverified.

## 0. Answer

game.exe's handler of client packet 0x41 (FUN_0054c0e0) sets the player's life, mana and stamina (stats 6, 8, 0xa) to the maxima of stats 7, 9 and 0xb, to the exact raw 256ths value, no clamp and no rounding. d2d's `Fight::revive` set only life. Mana and stamina stayed as they were at death. D1 is proven by a differential: 390 of 408 cases differ before the fix, 0 after (seed 20261108: 0 of 1508 after).

## 1. Questions

| # | Question | Answer | Tag |
|---|---|---|---|
| Q1 | Which path is "resurrect"? | C -> S packet 0x41, handler FUN_0054c0e0 (table 0x6e0d18, 8 bytes an entry, entry 0x41 = 0x6e0f20; the dispatcher FUN_0054d750 indexes it at 0x54d772 / 0x54d7d4 / 0x54d80c). The dispatcher's jump table sends 0x41 to the arm at 0x54d79e: `cmp [unit+0x10], 0x11` (0x54d7a2), a mode other than 0x11 (dead, I for the name of the mode) returns 0 at 0x54d821. The handler checks again (0x54c107 `cmp [esi+0x10], 0x11`) and that the packet length is 1 (0x54c0e6). Nothing else calls FUN_0054c0e0 (no direct reference in the full disassembly, only the table word at 0x6e0f20). Already in upstream `network.md:88` and `net-packets.md:793`. | V read; S |
| Q2 | What is refilled, to what? | 0x54c15c..0x54c179 read the maxima: FUN_00625d10 (stat 7), FUN_00625d60 (stat 9), FUN_00625db0 (stat 0xb), each the full stat list getter FUN_00624f60 with the stat in the high word of edi (`mov edi, 0x70000` at 0x625d4a). 0x54c17c, 0x54c18a, 0x54c198: `FUN_00627260(unit, 6 / 8 / 0xa, value, 0)`. No comparison, no `min`. Items and states count because the getter is the full list (base plus the modifier lists), the same getter the loader's stamina line uses (fid-loadhp-trace Q5). | V read; V run (base list: below) |
| Q3 | Run for real? | The unit is built by the oracle with the six stats; FUN_0054c0e0 runs as game.exe has it, with the real FUN_00625d10 / 25d60 / 25db0, FUN_00624f60, FUN_00627260, FUN_00625480. Result for 408 cases: life, mana and stamina equal the maximum in all 408, even for max 0 (all become 0, no floor of 1), fractions kept (`frac` case: 23373, 11619, 26125), saved values above the maximum are lowered to it. A control run with the unit's mode 1 (alive): 0 of 408 changed (the handler does nothing). | V run |
| Q4 | Other state changed in the handler? | After the three sets: FUN_00548520 sends 6, 8, 0xa to the client (0x54c19d..0x54c1c8); `or [unit+0xc4], 2` (0x54c1cd); state 0x36 (States.txt row 54, `uninterruptable`) set on then off through FUN_00639db0 (0x54c1d4..0x54c1e3; I: a way to run the state's callbacks, the net effect is none); FUN_0056dfa0 (0x54c157) walks the unit's inventory list (unit+0xa8) and re-applies each item's state through FUN_00639db0 / FUN_00646d60 (I: the states items grant, since the death took them); the unit goes to a level FUN_0053aec0 (0x54c216) picks through FUN_00620bb0 -> FUN_0061a1b0 -> FUN_006427f0 -> FUN_0061ab70 (I: the act's town, upstream `combat.md:494` says "to the act's town at its start"), FUN_005809d0 (0x54c229, I: sets the town mode); the two hand slots are re-equipped through FUN_00620190 / FUN_006201d0 and FUN_005701b0 (0x54c22f..0x54c286, I). A branch at 0x54c12c (`FUN_00538670` tests word [slots+0xa] & 4) calls FUN_0052caf0(.., 3) and returns 0 (I: the hardcore branch, upstream `combat.md:494` "hardcore ends the game"). | V read; I for the meaning of each helper |
| Q5 | Curses, states, gold, corpse? | Not in this handler (V read: no gold, experience, corpse or curse call in FUN_0054c0e0's 438 bytes). The death penalty runs when the death starts (FUN_00580ec0 -> FUN_00535ab0: experience, gold; upstream `combat.md:470..487`, S) and the corpse when the death has played (FUN_00581020 -> FUN_0057fca0). What clears the player's states at death is not decoded here. | V read for "not here"; U for where the states go |
| Q6 | Other paths that put a player back on its feet? | A merc is revived by its town NPC (docs/d2d-probes.md, never seen live), a different path and a different unit; the claim does not cover it. No Resurrect skill for the player was found in the table of packets (only 0x41). d2d's co-op ghost raise (Resurrect channel, `World::raise_step`) has no game.exe counterpart: upstream `deviations.md:82` lists it as a deliberate d2d feature. | S; U for "no other path" |

## 2. d2d side (V read)

| Caller | File:line (upstream main 9c78f391) | What it is |
|---|---|---|
| `Fight::revive` | `components/game/fight/setup.cpp:120` | before: `stats[kLife] = stats[kMaxLife]`, then it clears `regen`, `stamina_pot`, `potion_states`, `charges`, `self_states`, `curses[0]`, `marks[0]`, `stunned`, `pmode = -1`, and puts the merc beside the player |
| `World::respawn` | `world/setup.cpp:443`, called by `World::tick` at `world/tick.cpp:327` on `cmd::Resurrect` | the packet 0x41 equivalent (moves to the act's town, then `fight.revive`) |
| `World::revive` | `world/setup.cpp:87`, from `raise_step` (`world/setup.cpp:~120`) and `tick.cpp:564` | co-op ghosts (a partner's channel, or the party in a town): d2d's own mechanic, no game.exe counterpart (S, deviations.md:82) |

Comments in d2d already say what the intent is: `World::revive` has "full life and mana" at `world/setup.cpp:95`, `world.hpp:763` "with full life" and `fight.hpp:487` "full life, no potions or poison working". The code did life only. Mana is not regenerated while dead (`tick.cpp`: `if (!fight.dead()) fight.apply_regen`), so mana stayed at the death value. The next `Fight` refresh keeps `current = min(current, maximum)` (strike.cpp:190), so the refill to the maximum is stable.

## 3. Deviations

| # | d2d | game.exe | Tag | Differential |
|---|---|---|---|---|
| D1 | respawn refills life only; mana and stamina keep their value at death | life, mana, stamina := maxima of stats 7, 9, 0xb (0x54c17c, 0x54c18a, 0x54c198) | V read, V run: 390 of 408 differ before the fix, 0 after; second seed 0 of 1508 after | yes, built: `tools/emu/fr_revive.py` against `test_game --revive-stats` (fork branch, bot copy in this repo) |
| D2 | `Fight::revive` clears regen, potion states, curses, marks, stun; the maxima inside the refill still carry Battle Orders' and an aura's share until the next refresh | the handler clears none of these; where the death removes states is not decoded | U | could be built: run FUN_00580ec0 / FUN_00535ab0 and the mode-0 transition on a unit with states and read the state list (which code; oracle = game.exe's own list) |
| D3 | the co-op ghost revive (`World::revive`) and the respawn share `Fight::revive`, so the fix also refills a raised ghost's mana and stamina | no such path | I (a design choice, not a game.exe fact) | none possible (no counterpart) |
| D4 | gold / experience penalty at `kModeDT` (`death_penalty`), corpse at `kModeDD` | upstream `combat.md:470..487` | S (not re-run here) | exists upstream (death penalty traces) |

## 4. Bot findings (described, not changed)

- B1: `internal/bot/death.go` waits for the respawn and then heals/restocks; docs/combat.md:75 records the live respawn as "FULL life and mana (40/40)" on game.exe (V live, earlier drill). On d2d before this fix the bot would have come back with the mana it died with, so any recovery step that assumes full mana after a respawn behaves differently on d2d. The bot reads the unit and does not assume; no change is needed. U: whether any bot threshold (mana potion, `chicken`) keys off the first snapshot after a respawn.
- B2: stamina is not in docs/combat.md:75; after the fix both builds return with full stamina, so the bot's run/walk choice after a respawn matches (I).

## 5. PR candidate: yes

Branch `local/fid-revive-refill-r1`, commit `c3dac20a`, 4 files (components/game/fight/setup.cpp, components/game/fight.hpp, tests/game/test_game.cpp, tools/emu/fr_revive.py), one commit. Details and honest limits: [pr-revive-refill-draft.md](pr-revive-refill-draft.md). Not verified by an independent verifier; not opened.

## 6. Runs (V run, worker host, nice 19, `-j2`, GCC Release, own build dir)

```sh
# FR_TREE = the synced scratch clone of the fork branch (not the dev checkout); fr_build.sh builds test_game, fr_diff.sh diffs game.exe (fr_revive.py) against d2d
<worker host runner> sh tools/emu/fr_build.sh
<worker host runner> sh tools/emu/fr_diff.sh after          # cases 408, mismatching 0
FR_DIFF=<path of fr_diff.sh> <worker host runner> sh tools/emu/fr_before.sh   # swaps in the parent's fight/setup.cpp: 390 mismatching; full test_game on the parent aborts rc 134
<worker host runner> sh tools/emu/fr_seed2.sh                # seed 20261108: 1508 cases 0 mismatching; the alive control changes 0
```
Results: before 390 of 408 differ (the 18 that agree already had mana and stamina at the maximum); `test_game` on the parent aborts with `Assertion got[0] == 90 * 256 + 77 && got[1] == 40 * 256 + 99 && got[2] == 80 * 256 + 13 failed` (test_game.cpp:1304), rc 134; with the fix `test_game` rc 0, 29 `OK` lines, among them "OK: a respawn refills life, mana and stamina"; the oracle: 0 of 408 and 0 of 1508 differ. `pr_guard.py up/main HEAD`: 0 FAIL, 0 WARN. `presubmit.py --quick`: clang-tidy ok (no finding on an added line), the `gcc` step FAILED only on `apps/launcher/main.cpp` (`main.moc` missing: build dir configured for `test_game` only; not a touched file); `-fsyntax-only -Wshadow -Werror -Wall -Wextra` on fight/setup.cpp and test_game.cpp: rc 0 (V run).

## 7. Limits of the proof

- The oracle unit has no gear and no state: its stat list is a base list, so the full-list getter returns the case's maximum. That a worn item's or a state's bonus enters the maximum is V read (FUN_00624f60 is the same getter FUN_00625480 uses), not V run in this oracle; d2d's `kMax*` already is the sum (strike.cpp:171..193).
- The helpers around the three sets are hooked (slot object, item state re-apply, client updates, move to town, hand slots); the oracle proves the stat vector, not the town move or the mode.
- Only packet 0x41 is claimed; the hardcore branch, the penalty, the corpse and the states are not.
- d2d's co-op ghost revive shares the function (D3).
- No live read of game.exe's respawn with a hurt-mana character (docs/combat.md:75 has "40/40" without saying what the mana was at death).

## 8. Drill recipe (fork lab binary, U to confirm later; nothing run here)

On the lab binary (docs/forkdbg.md, docs/forkcheats.md; one hidden slot, offline `--no-save`, throwaway character, every call through `d2dctl.SkillDebug` / `d2drill -probe` with a reason):
1. `debug stat 8 256` (mana 1.0), `debug stat 10 256` (stamina 1.0), read `dump stats` (raw[8], raw[10]).
2. `debug hit 99999` (dies; wait for DD), `cmd resurrect`, `step 60`; read `dump stats`: expect life, mana, stamina = their maxima (after the fix; before it mana and stamina 256).
3. The same on game.exe live (a separate card, lab rules): read the server unit stats 6, 8, 0xa before the death and after Esc; they should be the maxima.
4. For D2: hurt the character, cast Battle Orders or a potion state, die, respawn, and read the state list (`debug states`) on d2d and the unit's state list on game.exe.

## Matrix cells proposed
No matrix doc covers death or respawn. Proposed row for docs/research/d2d-fidelity-index (or the next index): "respawn after death: life, mana, stamina to the maximum; d2d life only; fixed in candidate fid-revive-refill (verifier pending)".

## Missing glossary terms
"Respawn (packet 0x41)" as distinct from "revive" (d2d's function name, also used for a co-op ghost raise and a merc's resurrection).

## Shared-file lines for the merger
See `backlog/merge-lines/agent-fidrevive.txt`.
