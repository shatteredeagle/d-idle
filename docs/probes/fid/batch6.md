# Decomp batch 6 (agent decomp10, 2026-11-11): summary

> status: finished, offline (emulator on the worker host; no game, no live d2d, nothing pushed or posted, no fork branch). Cards: [fid-potions](fid-potions.md) (ST05), [fid-stamina](fid-stamina.md) (ST08), [fid-identify](fid-identify.md), [fid-gold](fid-gold.md), [fid-zeal](fid-zeal.md); task TASK-327. Scripts `tools/emu/d10_*` (V run: `d10_stam.py`, `d10_pot.py`, `d10_roll.py`; builder `d10_build.sh`; d2d sides `d10_stam_test.cpp`, `d10_pot_test.cpp`; the d2d tree used is the scratch build `pr/sorcrest-d2d` of the worker host).

## Answers

**Potions.** Healing and mana potions work over time: hp1..hp5 give 30, 60, 100, 180, 320 life over 192, 160, 171, 192, 256 ticks (3.9 to 31 life per second for a Sorceress); mana potions 20..250 over 128 ticks; Amazon, Paladin, Assassin x1.5 life and mana, Barbarian x2 life x1 mana, Sorceress, Necromancer, Druid x1 life x2 mana (V run, FUN_0062a5d0 / FUN_0062a620). Rejuvenation is instant: 35 or 100 percent of max life and mana. A second potion joins what is left: the rate stays, the duration grows, so chain drinking does not heal faster (V read, D arithmetic). Double fill `(vit - 1) / 4` percent: d2d equals game.exe (40,000 native cases, 0 mismatch); the live "0 of 24" is not the roll (U: the debug stat path). No server cooldown (V read).

**Stamina.** Run drain RunDrain x 2 x (armor speed / 10 + 1) per tick outside town (20 or 15 for the Assassin; speed 10 body armors double it), regeneration `max >> 8` standing, `max >> 9` walking and only above 1.0, none running: d2d equals game.exe (80,000 cases, 0 mismatch). Walk 0.375, run 0.5625 subtile per tick (S).

**Identify.** Cain: free after the rescue, 100 gold per item before (all or nothing, whole pack, cube and worn items, one request); a tome loses one charge per item (V read); stash and belt are outside Cain's count.

**Gold.** Pile `ilvl + rand(5 ilvl)` (V run earlier, no difficulty term); pickup is capped at clvl x 10,000 purse, 2.5M stash (V read).

**Zeal.** Strikes are rolled back by Skills Param2 (100) after each hit: the next strike follows after the action frame (7 ticks, one-hand sword, IAS 0), d2d waits a full swing (15). Five strikes 43 vs 75 ticks.

## Deviations table

| id | Card | game.exe | d2d | Tag | Differential | PR |
|---|---|---|---|---|---|---|
| D1 | zeal | swing restarts after each strike (Param2 100), spacing = action frame | full swing between strikes | game V run (fields I), d2d V read | to build (TASK-330) | no |
| D2 | potions | double fill (vit-1)/4 percent | equal in code (V run, 0/40,000); live 0 of 24 | U (debug stat path) | recipe R1 | no |
| D3 | identify | count: inventory, cube, worn | d2d also counts the belt | S vs S | buildable (FUN_0062a530) | no |
| D4 | gold | purse cap clvl x 10,000 on pickup | not read | game V read, d2d U | buildable | no |
| (equal) | stamina, potion arithmetic | | | V run, 0 mismatch | built | n/a |

No PR candidate: the only proven-in-game deviation (Zeal spacing) has no d2d-side run yet; a PR needs mismatches before and zero after.

## Bot findings and cards (proposals; nothing changed)

1. TASK-328 `potiondrink` (Proposed): count the pending heal; stop re-drinking healing potions while the last one works (`internal/combat/fighter.go:446`, `internal/resource/potion.go:79`); fix the wrong comment at `internal/env/tuning_combat.go:25`.
2. TASK-329 `runstate` (Proposed): read run or walk from the exact stamina steps (`internal/d2dinput/run.go`).
3. TASK-330 `zealdiff` (Proposed): the d2d side of the Zeal differential; Paladin Zeal kill-time models are conservative by 1.7 on d2d (fid-zeal.md B1).
4. Identify (consumers TASK-298, TASK-307; no new card): Cain's count and price in the planner (fid-identify.md B1..B4).
5. Gold: no change; do not add a purse-cap rule (unreachable at the bot's economy).

## Not done

Zeal d2d side; the potion state merge on the game side (needs the stat-list machinery); the unidentified-at-creation rule; Dragon Talon and Fend spacing; walk and run speed was not re-run (S, upstream V run). The worker host load was 13 to 15 on 16 cores during the runs; one heavy job at a time.

## Method notes

`tools/emu/run-on-worker.sh` with `D2_LABRUN_HOST` set; `d10_dis.sh ADDR:COUNT` for disassembly; hook pattern as `covt_regen.py` (hook the stat getters and setters, run the real function). The Zeal harness shows how to run a scheduler function without a world: hook the event add and clear calls and read what it schedules.
