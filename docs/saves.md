# Saves: integrity and progression from the .d2s (agent SAVES)

Code: `internal/saves` (new), uses `internal/cheat/d2s` unchanged (no API additions were needed).
Tags: V = read from a real file (Botty.d2s/.map 958 B and Testy.d2s 939 B, copied read-only from the worker host 2026-10-28, never committed),
S = published format / recollection, U = guess. Live results of 2026-10-28 (agent LIVEQ, Multi on i5, section "Live results" at the end) flipped the rows below that say V live.

## API
- `saves.Snapshot(saveDir, backupDir, name, Options{Game|AtMenu, Keep, Days}, now)`: copies `name.d2s/.key/.map/.ma0-9` to `<backupDir>/<name>/<UTC stamp>/`.
  Refuses (fail closed) when `GameState.InGame()` is true or errors, or when neither `Game` nor `AtMenu` is given; refuses a corrupt source .d2s; skips
  when .d2s and .map equal the newest snapshot; detects a size/mtime change during the copy; stages in a temp dir and renames. Retention: newest `Keep`
  (10) plus the newest of each of the last `Days` (14) UTC days.
- `saves.List`, `saves.Prune`, `saves.Restore(..., id "" = newest)`: restore verifies the snapshot, keeps the current files in `replaced-<stamp>/` (never pruned), same in-game gate.
- `saves.Verify(bytes)` / `VerifyFile`: signature, version 0x60, size field, checksum, section markers, stat block (`d2s.Parse`). The item section is not parsed.
- `saves.Decode(bytes)` / `ProgressFile` / `ProgressOf(*d2s.File)` -> `Progress{Name, Class, Level, MapID, Progression, Difficulty[3]}`; each difficulty has `Unlocked`, `LastPlayed`, `Act`, 39 `Waypoints` (bit, levels.txt id, name, Active), named `Quests` with raw word and flags, `RewardsEarned()` using the ids of `docs/plan/data/quest-points.json`.
- `saves.MapFileSeed(.map bytes)`.
- Paths are a stub (`saves.Paths`, `DirPaths`) until WP-STATE lands; `GameState` is supplied by the caller (mem/env/launch), e.g. `saves.GameFunc`.
- Not wired into anything: the sentinel/bot should call `Snapshot` before every game start and after each Save and Exit (at a menu), and `Verify` on load.

## What is verified vs not
| Claim | Tag |
|---|---|
| signature 0, version 4, size 8, checksum 0xC rotl+byte, name 0x14, level 0x2B, section markers (`Woo!` 0x14F, `WS` 0x279, `gf` 0x2FD) | V (Botty, Testy; also earlier by CHEAT on Multi) |
| Map id: 4 bytes LE at 0xAB (Botty 0x08ec30b2, Testy 0x58d768c6, differ per character) | V position |
| `<name>.map` is 24 bytes `0c000000 01000000 <seed> 00..`; seed equals the d2s header map id | V (Botty); other 20 bytes not understood (U) |
| Map id stays the same across Save and Exit and re-entry on the same difficulty (15 loads, with act/quest/waypoint edits in between) | V live |
| Map id changes when the game is entered on a different difficulty than the last saved one (3 of 3: Normal->Normal-with-Nightmare-flag, ->Nightmare, ->Normal) | V live; a second game without exiting (create-join): U |
| Quest block: 3 x 96 bytes after a 10-byte header (`Woo!`, u32 version = 6, u16 size) | V header (version 6 on Botty); word = u16 |
| Testy normal words 0 = 0x0001, 1 = 0x000c (started + left town) | V raw bytes; reading them as intro + Den of Evil is consistent, not proof |
| Quest flag bits: 0x0001 and 0x8001 both show "completed in a previous game" in the log (so `Done` = bit 0 or 15 is right); 0x000c shows Sisters as started (log text); 0x0010 alone shows Search for Cain active; akara "Den of Evil" talk wrote 0x0004 (bit 2 started), leaving town to Blood Moor made it 0x000c (bit 3), real saves show 0x0010 (Navi, entered the Den without talking) and 0x001c (Booti: started+left+entered area); the game ORs 0x1000 into a completed word once the log was opened (0x0001 -> 0x1001, 0x8001 -> 0x9001) | V live |
| Quest word index: act I words 1 Den, 2 Sisters, 3 Tools of the Trade, 4 Search for Cain (log icon order is Den, Sisters, Cain, Tower, Tools, Slaughter, the file order differs); act II word 10 Horadric Staff; act III word 20 Golden Bird; act IV word 25 Fallen Angel; act V words 37 Prison of Ice and 40 Eve of Destruction | V live (a word set by hand lit exactly that quest in the log) |
| Remaining quest words of acts I-III in the saves table order (5-6, 9, 11-14, 17-19, 21-22) | S (follow from the verified neighbours, not each lit) |
| Act V quest words are 35-40 (Siege 35 ... Eve 40), NOT 33-38 as this package first had: V live, table fixed. Act IV words 26-27, act V words 35, 36, 38, 39 in order | S. Intro/act-done words (0, 7, 8, 15, 16, 23, 24, 28, 32): positions U (the game keeps what the cheat writes, 0x8001) |
| The quest log shows the Act V tab only once Terror's End (word 27) is complete; with it absent, an act-5 start still loads Harrogath | V live |
| `Done` = reward granted or completed-before bit | U |
| Waypoint block: `WS`, u32, u16 size, then per difficulty 24 bytes (2 header + 5 bytes of bits); bit 0 = Rogue Encampment on fresh chars | V (Botty, Testy) |
| Waypoint bits 0 (Rogue Encampment), 1 Cold Plains, 3 Dark Wood, 8 Catacombs Level 2 | V live (panel rows and real Cold Plains touch gave exactly bit 1 = 0x3). The act-town bits the cheat sets (9, 18, 27, 30) survive the game's Save and Exit; their panel behaviour was not opened (U). Remaining bit names/ids: S |
| `none` mask still lists the town in the panel ("No other waypoints activated"); all 39 bits give all act-1 rows | V live |
| Difficulty bytes 0xA8-0xAA, bit 7 = last played, low bits = act (0 = act I on fresh chars) | V live: Normal 0x80..0x84 = acts 1-5 each loaded the right town; Nightmare picked on the difficulty screen wrote 00 80 00 |
| Difficulty unlock: progression byte (0x25) >= 5 makes the difficulty screen offer Nightmare | V live (screen appears, Nightmare picked, loads in Rogue Encampment). Eve of Destruction (word 40) done makes the game itself set progression 5 | V live. >= 10 for Hell: U |

## 10-minute live checklist (main agent, one throwaway char on a free instance, game offline)
1. Map id stability (F10, 3 min): `Snapshot` (at menu), enter game, Save and Exit, run `ProgressFile`; enter again, Save and Exit, compare `MapID`; make a second game without exiting-and-reentering (create-join) and compare; also compare `.map` seed. Stable = route caching possible.
2. Kill -9 loss (F3, 3 min): note the .d2s hash and level/exp, gain XP (kill one monster), `kill -9` the game, compare the .d2s mtime/hash: expected unchanged (so a kill loses everything since the last Save and Exit). Then `Restore` and check the game loads.
3. Quest bits (4 min, pick cheap ones): talk to Akara and take Den of Evil (word 1 should gain started), clear the Den and turn in (bit 0 reward granted, check `RewardsEarned`), then Sisters' Burial Grounds start; diff `Quests(0)` before/after each step to confirm word indexes. For acts III-V use `d2cheat save ... difficulty/act` is NOT enough; confirm order the first time a real character finishes a quest there.
4. Waypoints: touch Cold Plains waypoint, Save and Exit, expect bit 1 only.
Dead ends: none yet.

## Live results (agent LIVEQ, 2026-10-28, i5 Multi, game offline, all edits through d2cheat with a logged reason, Multi.d2s restored byte for byte at the end)
Checklist 1 (map id): stable across Save and Exit and re-entry on one difficulty, and across act-start, quest, waypoint edits; changes whenever the difficulty differs from the last saved one (see table). `.map` seed equals the d2s map id (V Botty earlier).
Checklist 2 (kill -9, `cmd/d2saves`): gained XP (5,000,000 to 20,000,000 in memory), `kill -9` Game.exe: the .d2s hash and mtime were unchanged, the XP was lost. `saves.Snapshot` at the menu, then an edit, then `saves.Restore`: hash identical to the snapshot, the game loaded it (exp 5,000,000). V.
Checklist 3 (quests): above; real talk to Akara (Talk, then Den of Evil entry) wrote word 1 = 0x0004 with word 0 untouched (Introduction was not clicked), leaving to the Blood Moor made 0x000c. Clearing the Den and turning in was NOT run (U: which bits the game writes at clear and at reward; 0x8001 shown as complete).
Checklist 4 (waypoints): reset to none, walked to Cold Plains with `navlive --god toarea 3`, activated its waypoint, Save and Exit: mask 0x3 (bits 0 and 1). V.
Tools added: `d2cheat qdump FILE` (header, map id, non-zero quest words, waypoint masks), `cmd/d2saves` (snapshot/list/restore/verify), `oog "start Name nightmare"`.
