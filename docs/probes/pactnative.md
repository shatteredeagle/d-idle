# pactnative: do the native difficulty levers work offline on 1.14d? (TASK-170)

> status: probe result of 2026-11-08 (overnight, one hidden instance). Facts for the Pact design note `docs/research/pact-system.md`; makes no decision. Tags: V verified live (a command was run in game.exe 1.14d and the result read), T read from the extracted game tables of the install on the worker host (Patch_D2, d2exp, d2data), S read from the private fork's source or decompilation notes (not run), U unverified.

## Method and provenance

- Instance i5 (hidden, private display), a fresh Paladin `Pacti` created through the normal OOG flow (`oog "start Pacti paladin"`), zero setup writes: no cheat tool was started, no memory write of any kind (reads only; input only through the d2input helper). Logged in `docs/probes/test-characters.md`. A second throwaway `Pacthc` (hardcore) was created for lever 3.
- Tool: `cmd/pactprobe` (new, read-only on memory): `say TEXT` (Enter, WM_CHAR per character, Enter), `fight SECS` (leave town to the Blood Moor, melee Fallen with the left-click Attack, poll monster and player life every 40 ms), `hc NAME CLASS`. `cmd/mpqcat` prints a table from the MPQs (T values below). Raw logs: `docs/probes/pactnative/players-n{1,2,4,8}.log`.
- Commands: `D2_INSTANCE=i5 tools/live.sh pactnative pactprobe say /players 4`, then `... fight 100` (new game per N: `oog leave`, `oog "start Pacti"`, say, fight). `D2_INSTANCE=i5 tools/live.sh pactnative mpqcat /tmp/d2mpq 'data\global\excel\Levels.txt' <columns>`.

## 1. `/players N` offline

| claim | result | tag |
|---|---|---|
| the command can be typed with the repo's input helper | yes: Enter opens the chat line, `char X` (WM_CHAR) per character, Enter sends | V |
| game.exe accepts it in offline single player | yes: the game prints `Pacti: players set to 4` in the chat area (read from a window screenshot) for N=4; the same line for 8, 2 and 1 was not screenshotted but the effect below was measured | V (4), V by effect (1, 2, 8) |
| range 1..8 | 1, 2, 4, 8 tried; 3, 5, 6, 7 not tried; values above 8 not tried | V for 1, 2, 4, 8, U otherwise |
| does the setting survive leaving the game | not isolated: every run typed the command again after `leave` and `start`; whether a new game starts at 1 is U (S: the count is per game) | U |
| monster life scales | yes, monotone in N (below) | V direction, magnitude U |
| max life readable from memory | NO: the client unit reports every monster at 128/128 (stat 6 and 7 are a 128 scale on the client, as `internal/d2/units.go` suspected); real max life is not available from this table | V |

Measurement (Fallen, Blood Moor, Normal, Paladin level 1 to 3, left-click melee, 40 ms poll). Each kill is one Fallen; a "hit" is a visible drop of its life; the character leveled during the series, so the damage per hit is not constant but the first-hit kill share is a fair indicator.

| N | kills | hits per kill (mean) | killed by one hit | formula (n+1)/2 |
|---|---|---|---|---|
| 1 | 13 | 1.00 | 13 of 13 | 1.0 |
| 2 | 11 | 1.36 | 7 of 11 | 1.5 |
| 4 | 4 | 1.75 | 1 of 4 | 2.5 |
| 8 | 8 | 2.25 | 1 of 8 | 4.5 |

Reading: the number of hits needed rises with N as the life factor predicts, but at N=1 a Fallen already dies to a single hit, so the hit count is capped from below and hides the real ratio: do not quote 1.36 / 1.75 / 2.25 as the life factor. The ceiling effect means the true factor is at least what the hit counts show and consistent with 1.5 / 2.5 / 4.5. An earlier N=4 run with slow polling (about 0.5 s) merged hits and is not used. The exact factor is S from the decompiled game (below), not V from this probe.

- S (private fork notes `docs/research/re/monsters.md` and `drops.md`, decompilation of game.exe 1.14d; not run): monster life and experience get a "/players bonus" percent per player count, table 0, 0, 50, 100, 150 ... (that is life x (n+1)/2, XP x (n+1)/2). The same count caps NoDrop for item drops: `n = near + (players - near) / 2`, so drops improve with N. The count at spawn is stored on the monster (stat 0x64) so it is frozen per monster.
- Spawn timing (V by effect): the factor applies to monsters created after the command; areas entered after typing it get it, which is why each N was a new game.
- Damage to the player: NOT measurable here. Fallen hits were 1 to 4 life points at every N (logged drops: N=1 two hits of 2; N=2 one of 4; N=4 two of 2; N=8 seven hits of 1 or 2) and the sample is too small and too coarse. The design note's "damage does not scale" is therefore U. (Arreat says monster damage does not scale with players; not re-verified.)
- XP effect and item drop / no-drop effect: NOT measurable in a short run (U live, S from the decompilation above). Level-ups during the series (1 to 3) show only that XP flows.
- Elite chance: no champion or unique was seen in the Blood Moor at N=4 (all monsters flags 0x0 in the first scan listing), which matches the Normal row of Levels.txt (blank MonUMin/MonUMax for Blood Moor, section 3), so `/players` does not add elites there: V for that one scan, U in general.
- d2d: the fork's chat line (`apps/d2d/town/update.cpp`) sends the typed text as chat text (or to the net game) and has no command parser; no `/players` string exists in the fork (grep of the pal-d2d3 worktree, 2026-11-08). So on d2d `/players` is a no-op for the monster count; the monster count there is the number of co-op players in the game (`make_monster(... int(players.size()) ...)`, `Monster::players`). A Pact that needs `/players` on d2d would need a fork change (a settable count), which is an overlay/fork decision, not a game feature. S (source read, not run on d2d).

## 2. Nightmare and Hell numbers (DifficultyLevels.txt, T)

File `data\global\excel\DifficultyLevels.txt` (Patch_D2.mpq wins over d2exp), rows Normal / Nightmare / Hell:

| column | Normal | Nightmare | Hell | meaning (U for wording, T for numbers) |
|---|---|---|---|---|
| ResistPenalty | 0 | -40 | -100 | added to the hero's fire, cold, lightning, poison resist |
| DeathExpPenalty | 0 | 5 | 10 | percent of experience lost on death (Arreat: of the span to the next level; corpse retrieval returns 75 percent; U from there, not tested) |
| MonsterSkillBonus | 0 | 3 | 7 | monster skill levels |
| MonsterFreezeDivisor / ColdDivisor | 1 / 1 | 2 / 2 | 4 / 4 | freeze and chill duration cut |
| AiCurseDivisor | 1 | 2 | 4 | AI curse (Confuse, Attract, Dim Vision) cut |
| LifeStealDivisor / ManaStealDivisor | 1 / 1 | 2 / 2 | 3 / 3 | steal cut |
| UniqueDamageBonus / ChampionDamageBonus | 90 / 90 | 75 / 75 | 66 / 66 | percent |
| HireableBossDamagePercent / MonsterCEDamagePercent | 50 / 50 | 35 / 35 | 25 / 20 | hireling damage vs bosses, Corpse Explosion vs monsters |
| StaticFieldMin | 0 | 33 | 50 | percent life floor |

So the user-facing numbers are exactly: Nightmare -40 resist and 5 percent death XP loss; Hell -100 and 10 percent. The -5 percent and -40 of the card text are right; the Hell values are -100 and 10. This matches `docs/difficulty.md` (V tables). Not tested: that the penalty is applied in game (no Nightmare character exists on the lab).
Entry facts: the unlock rule (Baal killed in Normal opens Nightmare) is game logic, not a table value: U here (S in `docs/progression/difficulty.md`). The character level range for entering is not restricted by these tables.

## 3. Elite density table (T)

The three tables that decide elites in 1.14d are all read; `Levels.txt` is the per-area one (V by reading the columns):

- `Levels.txt` columns `MonUMin`, `MonUMax` (Normal, expansion rows share the columns), `MonUMin(N)`, `MonUMax(N)`, `MonUMin(H)`, `MonUMax(H)`: how many unique/champion packs an area makes; blank means 0. `MonDen`, `MonDen(N)`, `MonDen(H)`: monster density (ordinary monsters). `NumMon` and `mon1..`, `nmon1..`, `umon1..`: the monster types (umon: the types used for elites). `MonLvl1Ex..MonLvl3Ex`: monster level per difficulty in the expansion.
- `SuperUniques.txt`: the named bosses (Bishibosh, Bonebreak, Coldcrow, Rakanishu, Treehead, Corpsefire ...) with their own placement, independent of MonUMin/Max.
- The meaning "MonUMin..Max = number of champion or unique packs" is U (veteran knowledge and the Arreat description); the values below are T.

| area (row) | MonLvl (Normal / NM / Hell) | MonDen N / NM / Hell | MonUMin..Max Normal | NM | Hell | NumMon, mon1 |
|---|---|---|---|---|---|---|
| Blood Moor (Act 1 - Wilderness 1, id 2) | 1 / 36 / 67 | 520 / 520 / 520 | blank (0) | 4 to 5 | 7 to 9 | 3, zombie1 |
| Cold Plains (Act 1 - Wilderness 2, id 3) | 2 / 36 / 68 | 520 / 520 / 520 | 1 to 1 | 4 to 5 | 7 to 9 | 3, brute1 |
| Den of Evil level 1 (Act 1 - Cave 1, id 8) | 1 / 36 / 79 | 600 / 600 / 600 | blank (0) | blank (0) | blank (0) | 3, zombie1 |
| Cave level 2 (Act 1 - Cave 2, id 9) | 2 / 36 / 77 | 600 / 600 / 600 | 1 to 2 | 4 to 6 | 6 to 8 | 3, skeleton1 |

Reading: monster density does not change with the difficulty; the elite count does: from 0 or 1 in Normal to 4 to 5 in Nightmare and 7 to 9 in Hell in the open fields. This is the per-area elite lever for an overlay Pact ("more elites"); `MonUMin/Max` are the knobs. Note the name key `Act 1 - Wilderness 1` is the Blood Moor (id 2). Fallen row in MonStats.txt (T): Level 1 / 36 / 67, minHP 21, maxHP 61 (Normal), 25 / 55 (Nightmare and Hell); the HP columns are scaled by MonLvl.txt and the /players bonus, so they are not the live life.

## Recommendation for the Pact decision (no decision made)

- `/players N` works offline on game.exe and scales monster life; it is typeable with the existing input helper (about 3 s). Nothing reads back the real life, so a bot cannot verify the scale itself; it can only trust the command and the S table.
- It does nothing on d2d without a fork change; native `/players` Pacts would exist on game.exe only, like Hardcore, unless the user wants a fork setting.
- A Pact for `/players 2` costs a hero about 1.5x kill time (S factor; the hit counts here only prove it is monotone), `/players 8` about 4.5x on paper; the Fallen at level 1 was a one-hit kill at N=1 and took 2 to 3 hits at N=8. XP rises by the same factor (S), so the Pact pays in experience but also changes the loop's pace.
- The Hardcore flag (lever 3, below) also works.

## 4. Hardcore offline (AC 3)

- V: `D2_INSTANCE=i5 tools/live.sh pactnative pactprobe hc Pacthc paladin` created a hero with the hardcore box ticked (click at the `hardcore-checkbox` of `internal/launch/oog_table.go`), the game showed one yes/no popup (`popup-yes-no`, the flow clicked yes), entered the game in Rogue Encampment with life and mana at start values, no crash. The saved character status byte (offset 0x24 of `Pacthc.d2s`) is `0x24` against `0x20` for the softcore `Pacti.d2s`: bit 2 (hardcore) set, expansion bit set. Death on a hardcore hero was not tried (it would end the hero; not needed for the design question).
- U: what the popup says; that the checkbox toggle is reliable on every run (one run).

## Cleanup

Instance i5 stopped with `tools/game.sh stop i5`; no job or process of this task left. To bring it back: `tools/game.sh start i5 && tools/game.sh helper i5`, then `D2_INSTANCE=i5 tools/live.sh multi oog "start Pacti"` (the hero stands in town in Rogue Encampment, level 3, no panel open; `Pacthc` (hardcore) is level 1 in the same list).
