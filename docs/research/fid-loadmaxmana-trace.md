# fid-load-maxmana: the Paladin maximum mana at load, 19.5 against 19.0 (TASK-102)

> status: current (agent decomp5, Sonnet, 2026-11-08; card TASK-102, matrix row ST01). Result: **NO DEVIATION in d2d. The 19.0 is the bot's reader, not d2d.** No PR candidate. Earlier: [fid-loadhp-trace.md](fid-loadhp-trace.md) (Q5 left the cause open), [fidelity-probe-run1.md](fidelity-probe-run1.md) (the 3-of-3 reads).

Tags: V read (read in source or disassembly), V run (observed running: the real code in a built test binary or the emulator), D (data), I inferred, U unverified. d2d tree: upstream main 1453f853 (and e36922ea, the base of the lab binary used in run 1), built on the worker host. Addresses are game.exe 1.14d.

## Q1. Where does the "19.0" come from?

The run-1 and run-2 reads (fidelity-probe-run1.md, table "Life and mana on load") gave the Paladin (clvl 4) maximum mana 19.5 in the file and on game.exe and 19.0 on d2d, 3 of 3 loads each. Three candidates were open: a truncation of the fixed point in d2d's load, in its recompute with the worn items, or in the reader.

| Step | Finding | Tag |
|---|---|---|
| The save | Pally.d2s (stat block decoded by hand with the ItemStatCost widths): level 4, energy 15, max mana raw 4992 (= 19.5), mana 4992, 14 items, no stat points left | V run (script decode, not in repo) |
| d2d loads it | `test_game --load-d2s Pally.d2s` (a local mode: `parse_header/stats/items` as `apps/d2d/load.cpp`, then `World::enter`, then one `Fight::update_fighters`): max mana **4992 after `enter` and 4992 after the update**, current mana 4992; `panel.bonus` for energy and max mana 0 (the gear adds no energy) | V run, upstream main 1453f853 |
| The same on the lab binary's base | the same mode built on e36922ea (Oct 29, the base of the lab binary `d2d.lab3-labverbs` used in run 1): **4992**, 4992 | V run |
| game.exe's loader | FUN_0056b180 -> FUN_0056af80 keeps stat 8 and its maximum exactly (`fl_load.py`, row `pally_life`: 4992 -> 4992) | V run, fid-loadhp-trace.md Q2 |
| What the bot reads on the d2d side | `internal/d2dsrc/player.go:160` takes the pair `mana` from the `dump player` section, whose values are `Stats::fixed(id)` = `get(id) >> 8` (components/d2s/d2s_items.hpp `fixed`), i.e. whole points; `playerStats` (player.go:215) then rebuilds the vector as `int32(v) << 8`. 19.5 -> 19 -> **4864** | V read |

So the d2d unit holds 4992 (19.5) and the bot's d2d source shows 4864 (19.0) because it only ever sees whole points. Run 1's "raw 4864" is the reader's shift, not the dump's `raw` array (that array is `stats.values` unshifted, apps/d2d/devctl_dump.hpp, V read on branch lab6). The same reader rule explains why every d2d life/mana/stamina read of the probe was a whole number (91/91, 88/88, 45/45) and why only the Paladin's maximum mana, the one value with a fraction, differed.

Conclusion: **no d2d deviation; the matrix claim "d2d truncates the maximum at load" is retracted.** The AC of TASK-102 ("cause found in d2d with file:line", "reproduced in a unit test", "fold into a PR") cannot be met because there is no d2d defect; the cause is found (the reader, `internal/d2dsrc/player.go:215`) and is a bot finding.

What is NOT proven (U): the lab binary itself was not re-run (no game here). Its `dump stats` raw array would show 4992 if the above holds; the drill below checks it.

## Deviation list

None for this card. D3 of fid-loadhp-trace.md (current life/mana do not follow a later change of the maximum, FUN_0055b800) stays open there.

## Bot findings

B1 `internal/d2dsrc/player.go:160` and `:215` (`fixed`): the d2d side of the fidelity probe carries life, mana and stamina and their maxima as whole points shifted left by 8, so every fraction of the d2d unit is invisible to `cmd/d2fidprobe`. The probe can therefore never show a fraction difference on d2d; the load comparison needs the `dump stats` `raw` array (256ths) instead. Described, not changed (no Go edits in this task).
B2 `internal/fidprobe/sheet.go:172` (`load.*` floats from `p.Stats[6+i]/256`): correct for game.exe, a whole-point value for d2d, see B1.

## PR candidate

No PR candidate: there is no d2d deviation to fix.

## Drill recipe (to confirm the U)

Lab binary with the lab verbs (docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md); no game was run by this task.
1. Start the lab binary offline (`--no-save`, lockstep) on a copy of Pally.d2s (cold dir of the worker host), `dump stats`, read `raw[9]`: expect 4992, and `raw[8]` 4992. If it shows 4864, d2d does truncate in the live path and this trace is wrong (the `--load-d2s` mode would then be missing a step of `apps/d2d/load.cpp`).
2. Step 25 frames, `dump stats` again: still 4992.
3. Repeat with an item that adds energy (the char panel path of `panel_stats`: `sum[kEne] * mana_per_energy / 4`, character.hpp:250): D3 of playerdef-trace.md (a quarter point of the gear share is truncated) is the only place a fraction can be lost in d2d's max mana, and it needs gear energy; Pally.d2s has none (`panel.bonus` 0).

## Matrix proposals (docs/research/fidelity-matrix.md)

- ST01: "Paladin max mana 19.5 in file, game.exe and d2d (4992); the d2d '19.0' of run 1 was the bot reader's whole-point shift (`internal/d2dsrc/player.go:215`, `Stats::fixed` = `>> 8`); test_game --load-d2s on upstream 1453f853 and e36922ea (V run). Load life/mana: PR #242 merged (bbf06f0f)". State RO -> OD for the max-mana part.

## How to re-run

`tools/emu/d5_patch_loadd2s.py` adds the `--load-d2s FILE` mode to the scratch tree's `tests/game/test_game.cpp`; `tools/emu/d5_oldbuild.sh REV NAME` builds an old commit with it; `tools/emu/d5_build.sh` builds the current one. Run: `test_game --load-d2s Pally.d2s` with `D2_MPQ_DIR` and `D2_PATCH_INSTALLER` set (the user's own game files).
