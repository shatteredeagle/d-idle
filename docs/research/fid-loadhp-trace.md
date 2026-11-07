# Trace: life and mana at character load, d2d against game.exe (TASK-101, pilot)

> status: written 2026-11-07 by the fidfix1 agent. Rows EN04, ST01. Evidence of the deviation: [fidelity-probe-run1.md](fidelity-probe-run1.md) (D2 file reader, scenario 15). Tags: V live = observed on the running game.exe or d2d; V read = read in source; V test = unit test run on the worker host; U = unverified. NO differential against game.exe's own code exists for this question (section 4).

## 1. Questions

**Q1. Where does d2d set life and mana on load?** `World::take_character` in `components/game/world.cpp:1019` (fork main c36b77b1; the same function in upstream main sits in `components/game/world/setup.cpp`). Both entry points call it: `World::enter` (a new game, first player) and `World::join` (co-op joiner). After copying the save's stats it runs a table `kRefill` of (life, max life), (mana, max mana), (stamina, max stamina) and sets each current value to its maximum (V read, `world.cpp:1026-1030`). The comment says "D2's save and exit to heal"; the upstream commit that added it is 756bf206 (2026-09-27, message: "a save left mid-fight started the next game near death"). The claim about D2 is not backed by a game.exe trace in that commit (U for the author's source).

**Q2. What does game.exe do?** It keeps the saved current value, fraction dropped. 6 of 6 loads, two characters (V live, i5, read 1 to 2 s after `oog start`): Freshe life file 89.54 / game.exe 89 / d2d 91 (max 91); Pally 77.76 / 77 / 88 (max 88); mana Freshe 24.39 / 25 / 45; Pally 19.5 / 19 / 19. Reads are with 8 fractional bits, so "89" is 22784/256; the one to two seconds of regen are inside the read (Freshe mana 25 to 32 on unhurried reads). d2d: 3 of 3 loads full (V live, lockstep, no step).

**Q3. Is the original's rule "keep" or "keep, except ..."?** Not shown. The saves were hurt by at most 12 percent of life and 50 percent of mana. Unknown (U): saves at 0 life, a dead character (game.exe puts it in the dead state), stamina (both saves were at max: the d2d refill of stamina is untested), and whether a hardcore or non-rejoin path differs.

**Q4. Why is the Paladin maximum mana 19.0 in d2d (4864) against 19.5 (4992) in file and game.exe?** Cause NOT found (U). Candidate sites in the fork, none run: (a) `Fight::update_fighters` (`components/game/fight/strike.cpp` upstream, max/percent block) adds `want = panel.bonus[kMaxMana] * 256`, whole points, to the max and takes `item_max` off again, so a gear or attribute share in quarter points is cut to whole points; (b) the `panel_stats` attribute share `sum[kEne] * mana_per_energy / 4` is integer quarter points; (c) a d2s load rounding. The Paladin carries two worn items; the d2d item list is UNREADABLE so the share cannot be told apart. The pilot fix does not touch it (the unit test pins max mana 4992 through `take_character` only, which passes before and after). Card TASK-102.

## 2. Deviation

| # | d2d | game.exe | tag | differential |
|---|---|---|---|---|
| D1 | life and mana = maximum at load (and stamina) | saved current, fraction dropped | V live 6 of 6 each side | not built: see 4 |
| D2 | Paladin max mana 19.0 | 19.5 | V live 3 of 3 each side; cause U | no (needs the Paladin save as a unit-test input) |

## 3. Fix and test (pilot branch)

Private fork branch `pilot/load-life-mana`, commit `693f9def` on fork main c36b77b1, pushed to the private remote the same day. Not in this repo (GPL). `take_character` now: for life and mana, `value = min(value & ~0xff, max)`; stamina still refilled (Q3). Test added to `tests/test_game.cpp` (the `World` the suite already builds): a saved 89.54/91 life, 19.5/19.5 mana and 50/102 stamina go through `take_character`; asserts life 89*256, mana 19*256, max mana 4992, stamina 102*256, and a saved life above the maximum is clamped to it.

Harness: `tools/emu/fl_build.sh` (worker host, `labrun <worker> --detach -- tools/emu/fl_build.sh`; `BASE=1` reverts the one source file to the parent). V test: with the fix `test_game` prints "OK: a save's life and mana are kept on entering a game (fraction dropped)" and finishes OK; with `BASE=1` it aborts on `Assertion hero.character.stats.values[kLife] == 89 * 256 failed` (mismatch before, none after). The test input comes from the live reads, so it proves d2d does what the live game.exe did on those values, not that game.exe's code does so for every value.

## 4. What is missing for the proof standard

1. A differential: game.exe's own character-load code (the function that copies the d2s stat block into the player unit and the code between that and the first frame) run under `tools/emu` against d2d for a table of stat blocks (life from 0 to above max, with and without fraction, mana, stamina, a Paladin 4992 maximum), oracle = the stat vector after the load. Not started: the load path in game.exe is not yet identified in our docs (V read: none of the existing traces covers it; U which function writes stats 6 and 8 from the file). The live loads stay the only evidence for game.exe.
2. Stamina, zero life and a dead save (Q3) on game.exe; use the fixtures of TASK-105 (fresh characters only) or hurt-by-play saves, never memory writes as evidence.
3. The fraction drop: is it the load (V live: the value read 1 to 2 s later is an integer) or a first-frame normalisation; one more game.exe read at 0 s would separate them (U).
4. D2 (max mana) cause, TASK-102.

## 5. Bot findings

B1. The bot treats a restart as a heal? Not found in `internal/` (grep: no load-time full-heal assumption). The Gate and soak code counts "Save and Exit" (chicken) in `internal/d2dsoakstat/invariants.go:120,304` as reported, not gated; with the d2d deviation a chicken-and-rejoin on d2d returns full where the original returns hurt, so a chicken rate measured on d2d over-states survival (I). Described, not changed.

## 6. Drill recipe (fork verbs, to confirm U later)

Lab binary with `--lab-verbs`: set a Paladin to life 1, mana 0 (setup writes are fine as a mechanism test, tag CHEATED, never parity evidence), save and exit, reload on d2d and read `dump`; on game.exe load a character saved by genuinely playing hurt (or dead) and read stats 6 and 8 at 0, 1 and 3 s. Compare stamina the same way.

## 7. Matrix proposals

- EN04: add "root cause located: `World::take_character` refill (fork world.cpp:1026), added in upstream 756bf206 on an unproven claim; fix on private branch pilot/load-life-mana 693f9def, unit test fails before and passes after; differential vs game.exe code missing (TASK-101)".
- ST01: unchanged (V live); max mana cause open (TASK-102).

## Shared-file lines for the merger
See `backlog/merge-lines/agent-fidfix1.txt`.
