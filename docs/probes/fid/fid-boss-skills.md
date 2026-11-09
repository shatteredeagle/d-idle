# fid-boss-skills (TASK-86, matrix row MA06 neighbour): boss skills and death mods, d2d against game.exe

> status: finished 2026-11-09 by agent decomp7 (Sonnet), PARTIAL. Result: the boss **death handler (MonUMod 22 / 21)** oracle passes on upstream main 7557a54d (V run, game.exe's own code, self-checking `ok`), and the boss **level, life, defence, experience** oracle is 0 real differences (V run). The four bosses' **skills** (Andariel poison cloud, Blood Raven raise, Countess, Corpsefire) have NO differential: only the table side is read here (D). AC #1 and #2 are therefore NOT ticked; AC #3 is not done (needs a game).

Tags: V run, V read, D (tables), I, U. Worker host, scratch tree = upstream main 7557a54d. Script `tools/emu/d7_boss.sh`.

## Q1. A boss's death kills the monsters round it: oracle ok V run

`boss_deaths.py` (merged upstream, 8949cc77 and 10a4abff): dispatch of mod 22 by MonStats class, parameters (radius, delay, undead only; Radament's max from Missiles.txt row 347 Range; Mephisto's missile), the search area through FUN_0065aa40, the pick and the delay on the boss's seed, mod 21's event-2 hook per state. It checks game.exe's own values against WANT_* constants and printed `ok` on this head (also: mod 21 state 54 -> event 7 at +3; owner a player -> owner kill; owner a monster -> death). **The d2d side is tested by `tests/game` (the commit that landed with it), not by this script**: a direct game-to-d2d line diff was not run here (U); the script pins game.exe's side only.

## Q2. Boss level and stats: 0 real differences V run

`boss_level.py gen 100000 20261109` (136,624 lines: every MonStats row x difficulty x kind plus 100,000 random) against `test_boss_level`: 136,624 lines, 5,931 differ, **all 5,931 are one family**: field life256, game.exe under one point (e.g. 192 of 256) where d2d holds 256 (`Monster::lifeless` one point, upstream 5ca89a60: "a zero-life roll lives, any hit kills it"). I checked the whole list by script (game life under 256, d2d exactly 256, all other columns equal): 5,931 of 5,931. The family is the representation upstream chose; both die from any landed hit and differ only for a hit of 0 damage (I, as in fid-boss-mods.md). Same finding as decomp6 on seed 20261108.

## Q3. The four bosses' skills (tables D, no differential)

From monstats.txt of the 1.14d patch stack (`mmtab/monstats.txt`):

| Boss | AI | Skills (mode, level) | MissA / MissS |
|---|---|---|---|
| Andariel | Andariel | AndrialSpray (seq_andarielspray, 1), AndyPoisonBolt (A1, 1) | none |
| Blood Raven | BloodRaven | Nest (seq_bloodravencast, 1), Quick Strike (seq_brquickstrike, 1) | MissA1 raven1 |
| Countess, Corpsefire | superuniques: no monstats row of their own (the file has none under those Ids; they are rows of SuperUniques.txt over a base class) | their skills are the base class row plus the unique mods; not read (U) | not read |

d2d side pointers (V read of source, not compared): Andariel's spray `andarielspray` (ai.cpp:2235), her bolt through `andariel_missile` (ai.cpp:403, 2066), Blood Raven's Nest / raise through `MonAct::seq` (ai.cpp:1801). The monster record of the spray and the bolt is the same function as the MissA path of fid-missile-monster.md only for MissA rows; **skill-sourced rows go through `andariel_missile`, which `mm_diff.py` excludes by design**, so no existing differential covers them (U). To decompose them: game side `mm_launch.py` already lists "the srvmissilea of its skills" (docs header); add them to `mm_diff.py` cases (source = skill srvmissilea at the skill level) and compare with a d2d test that calls `andariel_missile`'s record builder; Blood Raven's raise and Corpsefire's skills need their srvdofunc read.

## Deviations (d2d against game.exe)

None proven. Open and unverified: skill-sourced boss missiles (U), Blood Raven's skeleton raise count and timing (U), the death mods other than mod 22 / 21 (U).

## Bot findings

None new.

## PR candidate

No PR candidate: nothing deviates in what was run.

## Drill recipe (lab, not run)

`debug spawn` Blood Raven with `debug states` on the player and read the raise count and interval for 60 s (game.exe vs d2d expects equal per Monster.txt "Nest" skill); Andariel's poison spray: spawn at range, record the poison length and per-frame damage on the player with `debug states`. Needs both lab binary and a game.exe instance.

## Matrix-cell update (proposed)

MA06: add "UPDATE 2026-11-09 (decomp7, fid-boss-skills.md): boss_deaths oracle ok and boss_level 136,624 lines with only the lifeless representation (5,931, explained) on 7557a54d (V run); boss skills (Andariel, Blood Raven, Countess, Corpsefire): table side only, no differential (U)."
