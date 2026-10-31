# Ranked live-verification list (progression)

Everything in this folder tagged U is resolved here. Ranked by *unblocked work per minute*. Commands follow AGENTS.md and docs/cheats.md / multi.md:
live runs go through `tools/live.sh <agent> <pkg> ...`, instances i1..i5 are hidden Xvfb games, cheats only on throwaway characters (Testy, Botty, Navi, Combi,
Towni, Booti, Multi) through `d2cheat --reason "..."`, screenshots only with `tools/shot.sh OUT.png iN` (game window only). Stop an instance you started with
`tools/game.sh stop iN` when done (agents never touch another agent's instance; the main agent lends one). Keep each live run under 3 minutes or use `--bg`.

Instance suggestion (the main agent decides): **i5 / Multi** for all save-edit work (already used for that, V cheats.md), **i1 / Navi** for map and preset dumps
(`navlive --god`), **i3 / Towni** for NPC dialogs, **i4 / Booti** for the first handler runs.

Result recording: add one dated line per check to `docs/progression/results.md` (new, by the verifier): id, date, instance, value seen, new tag (V/S/U). Flip the tag
in the file that cited it in the same commit.

Status 2026-10-28 (agent LIVEQ): O-1, V-02, V-03, V-08 (dialog/bytes) done and V, V-04 for Akara only, V-01 indexes confirmed by quest-log display. One line per check in results.md. Act 3 quest order (17 Lam Esen .. 20 Golden Bird) and act 5 positions (35-40) were corrected in quests.md.

## Ranking

| rank | id | what it unblocks | minutes | instance | needs |
|---|---|---|---|---|---|
| 1 | O-1 | quest block layout sanity on real saves (zero game time) | 10 | none | DONE 2026-10-28 (V, results.md) |
| 2 | V-01 | (indexes V live 2026-10-28 by hand-set words; clear/turn-in bits still U) quest bit meanings and index map: every `Need()` in every handler | 45 | i5 | none (plays Act 1 once) |
| 3 | V-02 | (DONE 2026-10-28, V) baseline flags: expansion bit, difficulty bytes, progression, act, merc readiness | 5 | i5 | none |
| 4 | V-03 | (DONE 2026-10-28, V; NPC ids still open) cheated start in Act 2/3/4/5 works; town waypoint state on arrival; NPC class ids per act | 15 | i5 | `act=` honoured |
| 5 | V-04 | (Akara only, 2026-10-28) dialog row counts and indices per NPC (Warriv, Meshif, Tyrael, Kashya, Greiz, Qual-Kehk, Larzuk, Anya, Malah, Cain, Atma, Jerhyn) and 0x11 clearing | 35 | i5 or i3 | V-03 |
| 6 | V-05 | memory quest/waypoint pointers (T2) | 30 | i5 | V-01 (known states) |
| 7 | V-06 | waypoint panel tabs of acts 2-5 and cross-act travel | 20 | i5 | V-03, waypoints=1 |
| 8 | V-07 | act hop A1->A2 end to end (Warriv) | 15 | i5 | Andariel bit or R1 |
| 9 | V-08 | (dialog + bytes DONE 2026-10-28, V) difficulty dialog, difficulty byte values, unlock | 15 | i5 | `progression=` edit |
| 10 | V-09 | merc: hire, unit owner field, revive, equip panel | 30 | i5 | V-03 (Act 2) + gold |
| 11 | V-10 | boss arena presets (Andariel, Duriel tombs, Countess, Summoner, Nihlathak) | 40 | i1 | `navlive --god` |
| 12 | V-11 | boss unit facts: superunique flags, Attacking2 mode, kill detect | 25 | i1 | V-10 |
| 13 | V-12 | Cairn stone order and quest-object modes (Orifice, seals, Compelling Orb) | 30 | i1 | god mode |
| 14 | O-2 | boss resistances and quest item codes from game data (zero game time) | 20 | none | `internal/data` |

## Offline checks (no game)

### O-1: read the quest block of real saves
1. Copy the saves at a menu (never while in game): `mkdir /tmp/progdesign-saves` on the worker host; copy `*.d2s` from each instance's `Saved Games/Diablo II` (Testy, Navi, Combi, Towni, Booti, Multi).
   (Cleanup: delete the folder afterwards.)
2. `hexdump -C -s 0x14F -n 288 <file>.d2s` for each. Expect the 4 bytes "Woo!" at 0x14F, then 6 more header bytes, then 3 x 96 bytes (check against
   cheats.md: "quests 'Woo!' at 0x14F (3 x 96 bytes)").
3. Also `D2S_REAL=<folder> go test ./internal/cheat/d2s -run Real -v` (round trip) in any worktree.
4. Expected: all bytes zero in Hell and Nightmare; Normal differs from zero in the characters that talked to NPCs (Testy: Akara/Gheed/Charsi were contacted, V town.md).
   Any non-zero WORD in a character that did nothing but talk is the "spoke to" or "greeting" bit: record index and value.
5. Record: first evidence of bit positions. This alone may settle indices 0-3 and the "started" bit.

### O-2: boss data and quest items from the game data
Write a 15-line throwaway `go test` (not committed) that prints, via `internal/data`: monstats rows for Andariel, Duriel, Mephisto, Diablo, Baal, Council
(three class ids), Izual, Radament, Summoner, Nihlathak, Countess, Blood Raven, The Smith, Ancients (3), Shenk: resistances per difficulty, `boss`, `superunique`,
`ai`, levels; superuniques.txt area; misc.txt codes for Horadric Malus, Scroll of Inifuss, Moldy Tome, Cube, Staff, Shaft, Viper Amulet, Jade Figurine, Khalim's parts, Soulstone,
Hellforge Hammer, Wirt's Leg, Lam Esen's Tome, Scroll of Resistance, Book of Skill; levels.txt id and name for every area id cited in bosses.md/acts.md.
Expected: the numbers in runs.json/areas.json (R) either confirmed or corrected. This turns the largest block of U in bosses.md into V and costs no game.

## Live checklists

### V-01: quest bit meanings (rank 2, 45 min, i5 Multi)
Goal: for 9 Act 1 states, the quest WORDs of Normal (bytes 0x14F+6.. ). Characters: Multi (clvl 25 by cheat, level gain is allowed). All actions in Normal.
Setup (each snapshot = leave game, copy `.d2s`, restart): `tools/live.sh multi oog leave`, copy `Multi.d2s` to `/mnt/cold/d2lod/progdesign/Multi-Sx.d2s` (bulky cold data, AGENTS.md), `tools/live.sh multi oog "start Multi"`. If WP-SAVES `saves.Snapshot` has merged, use it.
States and the action that makes each:
| snapshot | action | expected word change (candidate, U) |
|---|---|---|
| S0 | fresh start, nothing spoken | baseline |
| S1 | talk to Warriv, Akara, Charsi, Gheed, Kashya, Cain (first contact plays greeting, V Akara) | "spoke to" bits of index 0 and maybe 1 |
| S2 | accept Den: talk to Akara after S1 | Den (index 1) bit "started" (2?) |
| S3 | clear Den of Evil (clvl 25 + `god on --for 5m`; kill all monsters) but do NOT turn in | Den bit "reward pending"/"complete" (1/0?) |
| S4 | turn in at Akara (skill point appears: read stat 5) | Den "reward granted" (0?) and maybe 13 |
| S5 | kill Blood Raven (Cold Plains->Burial Grounds), no talk | index 2 changes |
| S6 | talk to Kashya | index 2 changes again |
| S7 | take Moldy Tome? (skip) -- instead kill The Smith (Barracks), do not turn in | index 3 |
| S8 | kill Andariel (Catacombs 4), not talking to Warriv | indices 6, 7 |
Steps: for each snapshot do: `d2cheat show` (UI flags, note 0x11), then leave and copy the save; analyse offline with a diff of the 288-byte quest block (`cmp -l`).
Expected: a table "event -> (index, bits)" that confirms or corrects quests.md section 2. Fill `QuestBits.Done/Started/Pending`.
Also check: does the UI flag 0x11 rise on S2 and S4, and does opening the quest log (`q`, flag 0x0F) clear it? (Writes into V-01 results.)
Cheat ledger: `level 25`, `god on` need `--reason "quest bit mapping for progression design"`.
Cleanup: character stays in town, panels closed; delete the cold-storage copies only if the user wants (they are cold data).

### V-02: baseline flags (rank 3, 5 min, i5)
1. `tools/live.sh multi d2cheat --reason "progression baseline" show`; note stat 12, difficulty byte at RVA 0x3A060C (expect 0), area.
2. From the Multi save copy: status byte 0x24 (expect bit 0x20 set = expansion; mercs need it), difficulty bytes 0xA8..0xAA, progression 0x25, act in 0xA8 low bits.
Expected: 0x24 & 0x20 != 0 (U; if 0 the character is classic and mercs/Acts 4-5 do not exist: stop the merc and Act 4/5 work for it); 0xA8 = 0x80|0 for Normal active in Act 1.
Record: the three bytes before any change, then after a leave-and-start (does it flip?).

### V-03: cheated start in later acts (rank 4, 15 min, i5)
1. `tools/live.sh multi oog leave`
2. `tools/live.sh multi d2cheat --reason "act start test" save Multi level=25 gold=100000 waypoints=1 difficulty=0 act=2` (parameter spelling from cheats.md)
3. `tools/live.sh multi oog "start Multi"`; `tools/live.sh multi d2cheat --reason "act start test" show`.
Expected (U): `Player.Area == 40` (Lut Gholein), `act` stat 1 (zero based), waypoints active include Lut Gholein's.
4. Record: town waypoint bit on arrival (does `waypoints=1` already include the town?); list NPC class ids in the unit table (`townlive` or `navlive town` print); screenshot (`tools/shot.sh /tmp/progdesign-a2.png i5`) to confirm.
5. Repeat for `act=3` (area 75), `act=4` (103), `act=5` (109).
Pass: all four arrive in the right town. If `act=` is ignored or crashes the load: record it, the fallback is R1 (quest bits, packages.md) plus a playthrough.
Cleanup: leave the character in the last town; `oog leave`, set back `act=1` before handing back.

### V-04: dialog rows per NPC (rank 5, 35 min, i5 after V-03)
For each NPC in the act you stand in (Act 1: Warriv, Kashya, Cain; Act 2: Meshif, Jerhyn, Greiz, Atma, Drognan; Act 3: Hratli, Ormus, Alkor, Asheara; Act 4: Tyrael,
Jamella; Act 5: Qual-Kehk, Larzuk, Malah, Anya): use the TOWN2 `pickEntry` machinery (town.md) or `navlive`/`townlive`, `GoToUnitOf(Monster, id)`, `Interact`, wait for UI 0x08,
then `tools/shot.sh /tmp/progdesign-<npc>.png i5` and read the list. Do NOT click anything but Cancel.
Record per NPC, in the quest-bit state at hand: row count, row index from top of each option (Talk, Trade, Hire, Resurrect, Go East, ...), Cancel last (V town.md).
Expected: Cancel is the last row; rows 15-16 px apart (V). The travel row for Warriv/Meshif exists only after the quest bit (check both states with V-01 saves).
Also: first contact greeting (speech box) vs menu on the second contact for each NPC (V Akara). Write rows into `internal/town/dialogtable.json` shape (acts.md section 3).
Delete screenshots after transcribing.

### V-05: memory quest and waypoint pointers (rank 6, 30 min, i5, uses V-01 states)
Run `docs/goals/tracking.md` spike items 1-7 in this order of cost: (1) dump PlayerData +0x10..+0x30 and look for 3 plausible pointers then 3 more (D2BS layout, U);
(2) dereference each, dump 96 WORDs / 24 bytes; (3) with the V-01 states re-do two transitions in game (talk to Akara; kill Blood Raven) and compare the dump with the V-01 save diff.
Pass: the in-memory WORD at index 2 equals the saved WORD after the same actions. Canary: record pointer chain as `PlayerData+0x?? -> +0x?? -> [96]uint16`.
Fail: record what the 6 dwords really are; T2 stays U and T1 (save) remains the quest source with the 0x11 edge as trigger to snapshot at the next menu.

### V-06: waypoint tabs for acts 2-5 and cross-act travel (rank 7, 20 min, i5)
Setup as V-03 with `waypoints=1`. Per act: stand at the town waypoint (`GoToUnitOf(Object, waypoint)`; class 119 in A1, others from V-03), `Interact`, screenshot; record
tab x positions (extrapolation: 173, 230, 287, 344 at y 77, nav.md), row y of the first row, row order. Click the tab, click a row, record area change time. Cross-act:
from Act 2 town click tab I and Rogue Encampment; expected an Act 1 arrival (U). Also click an inactive row: expect `ErrNoWaypoint` behaviour (nothing happens).
Cleanup: Esc to close; back in town.

### V-07: A1 -> A2 hop (rank 8, 15 min, i5)
Option A (preferred once R1 exists): prepared state: Andariel bits (6, 7) set via quest editor on Multi, `oog start`. Option B: play S8 of V-01.
Steps: `ToTown` if needed, `GoToUnitOf(Monster, Warriv)`, `Dialog.Open`, screenshot, pick the travel row by index recorded in V-04, wait for area change.
Expected: area 40 within 5-10 s. Compare `Player.Area`, then check the town WP and waypoint bits (V-05 pointer if available).
Record: loading time, whether a speech box appears on arrival (it can block clicks, V Akara), whether the character's act byte changes.

### V-08: difficulty dialog and byte (rank 9, 15 min, i5)
1. Fresh Multi: `oog leave`, `save Multi progression=<N>` with several N to find which unlocks Nightmare (cheats.md unverified), start, screenshot the dialog before choosing.
2. Observe: does the dialog appear? Which buttons enabled? Rects as in difficulty.md (Kolbot).
3. Click Nightmare; read the difficulty byte (RVA 0x3A060C): expect 1 (U); area 1; resists lower by 40 (read stats 39/41/43/45).
4. Leave and start again: does the dialog appear a second time, which difficulty is preselected?
5. Save and Exit button geometry from the Esc menu (classic client, 800x600): screenshot at the menu, record the click point of "Save and Exit Game".
Pass: all three difficulty byte values known, dialog behaviour recorded. Cleanup: restore `progression`/`difficulty` of Multi to Normal (cheat ledger entry).

### V-09: mercenary (rank 10, 30 min, i5 after V-03 at Act 2)
1. `gold=100000` save edit, Act 2, stand in Lut Gholein. Talk to Greiz: screenshot the dialog and the hire list (rows, candidate count, aura text, cost).
2. Hire the cheapest candidate; verify: gold drop equals quoted cost; a friendly monster appears near; dump the unit: class id, owner field candidates (look for the player's gid
   in the unit struct dwords 0x00..0x100), level, skills.
3. Find the equipment panel hotkey or portrait click; screenshot; read UI flags before/after (new flag?). Equip a bought helm with `hand.Transfer`; verify by owner read.
4. Make the merc die: walk it into Dry Hills at clvl 25 without god (the character stays alive with invincibility only if `god on` is off; a merc dies, not the player, in at most a few minutes) or `kill` via the
   cheat if one exists; verify what changes in memory (unit gone? flag?).
5. Talk to Greiz again: the Resurrect row; revive; gold drop; unit back.
Pass: owner field located, dead flag located, revive cost formula noted (level times factor). Record the hire/revive costs in `mercs.json` evidence column.
Cleanup: dismiss nothing; leave the merc alive if the main agent allows; Multi stays in Act 2 town.

### V-10: random-layout boss arena presets (rank 11, 40 min, i1 Navi)
`tools/live.sh nav navlive --god ...` visits (logged cheat actor nav2, V). For each: Andariel (Catacombs 2 -> 4), Countess (Black Marsh -> Tower Cellar 5), Duriel
(Canyon of the Magi: all six tomb entrances; find the one with the Orifice), Summoner (Arcane), Nihlathak (Halls of Vaught), Radament (Sewers 3), Izual.
Use `d2mapdump` or `navlive exits/presets` to dump presets (type, class id, room coords) per visit; save under `/mnt/cold/d2lod/progdesign/presets/` and one summary table.
Record per boss: preset class ids (HoradricStaffHolder, SuperChest, Journal, NihlathaksPlatform, HoradricScrollChest), distance from the level entrance, whether the same
level in a *second* game of the same character is identical (feeds F10: map seed stable per character?).
Pass: each anchor class id known; list becomes the `Anchor` constants in bosses.md.

### V-11: boss unit facts (rank 12, 25 min, i1 Navi)
At each arena from V-10 with `god on --for 5m`: read the boss monster unit: class id, unique flags (+0x16/+0x1C/+0x26, tracking.md item 20), monster mode values during
his attacks (Mephisto `Attacking2`, Duriel melee), hp scale, death mode (12?) and `LifePct`. Expected: boss class ids match runs.json (R); death mode value; the superunique name index.
Pass: `Done(w)` predicate for Kill proven on at least Andariel and Mephisto.

### V-12: quest objects (rank 13, 30 min, i1 Navi)
Cairn stones (Stony Field): list the five stone objects and their `Mode` before/after touching in order; do two different games on the same character to see whether
the order repeats (answers execution.md open question). Chaos seals (Diablo) and Compelling Orb: `Mode` change after interact. Horadric Orifice: `Mode` before/after staff placement.

## Done criteria for the list

All of O-1, V-01..V-04 passing converts every `Need()` and every NPC hop to V and unlocks the step layer with real data; V-05 is the speed-up (T2); V-06..V-09 unlock Acts 2-5 and
merc code; V-10..V-12 unlock the boss scripts. Minimum useful first session: O-1, V-02, V-03, V-04 (one hour).
