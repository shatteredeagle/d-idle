# In-memory quest and waypoint data (agent MEMQUEST, V-05)

Code: `internal/d2/questmem.go` (readers), `internal/questmem` (a `quest.Source`), `cmd/d2qmem` (live probe, fixture recorder).
Tags: V = verified live on the worker host i5 (Multi, 1.14d, 2026-10-28), S = standard/recollection, U = unverified.

## Where the data is (V)
- `PlayerData` = `UnitAny+0x14`: `char name[16]`, then `QuestInfo*` Normal/Nightmare/Hell at +0x10/+0x14/+0x18, then
  `WaypointInfo*` Normal/Nightmare/Hell at +0x1C/+0x20/+0x24 (D2BS layout, confirmed). +0x2C reads 1 (meaning U).
- `QuestInfo` is `{WORD* buf; DWORD x}` (stride 0x20 in a table; x read 0x300, meaning U). `buf` points to the same 96 bytes (48 LE
  WORDs) as the save's quest block, same indexes (word 1 Den ... 40 Eve). Buffers are 0x80 apart.
- Waypoint struct: bytes `02 01` then the 5 mask bytes (little-endian, 39 bits, same bit order as the save's "WS" section), then zeros; stride 0x10.
- **The client player unit (the global `PlayerUnit` that `d2.Reader.Player` reads) has all these pointers but its buffers are all zero
  (waypoints only the town bit) whatever the save says. Only the embedded game server's player unit holds the real data.** The server
  unit has no static pointer (same finding as docs/cheats.md); it is a second `UnitAny` with the same id/class, an extended stat list
  (flag 0x80000000) and the same name. `d2.FindServerPlayer` finds it by scanning the private rw mappings (about 40-55 ms live, i5).
  The address changes per game start (0x3b92600, 0x3a42600 seen), it did NOT change on area changes in a game (Rogue Encampment -> Blood Moor -> Cold
  Plains -> back), but the Source revalidates and rescans anyway.

## Verification (V)
1. Fresh start of Multi: memory equalled `d2cheat qdump` of the save for all three difficulties (Normal word 0 = 0x0001, wp 0x7fffffffff, Nightmare
   and Hell empty, wp 0x1).
2. Distinct values written to the save at the menu (Normal sisters 0x1c; Nightmare den 0x0c, tools 0x8001; Hell radament 0x1001, word 35 = 0x0004;
   waypoints none): after loading, memory showed exactly those words in the right difficulty slots.
3. Live changes, no save in between: talking to Akara (greeting, Escape through the speech into the Den of Evil text) wrote Normal word 1 = 0x0004
   at once; walking out of the Rogue Encampment made it 0x000c; touching the Cold Plains waypoint made the Normal mask 0x3 (bits 0, 1).
   Memory before Save and Exit (words `1=0x000c 2=0x001c`, `wp 0x3`, Nightmare/Hell unchanged) equalled the saved file exactly afterwards.
   Reading while the area changed (toarea 1) never saw an inconsistent state in 100 ms polls: one scan only.
4. Fixture: `internal/questmem/testdata/quests-i5.snap` (12 pages, recorded by `d2qmem record=FILE`; game memory of a throwaway character,
   no save file). Replays with the test supplying a one-page region for the scan.

## quest.Source (`internal/questmem`)
- `questmem.New(*d2.Reader)`; `Word(diff, id) (uint16, bool)` implements `quest.Source`, `Evidence()` = `EvMemory`. Also `WordIndex`, `Waypoint(diff,
  levelsTxtArea) (active, known)`, `Progress()`, `Invalidate()`, `Scans()`.
- List it before `quest.SaveSource` in `quest.NewReader`. Interpretation stays `quest.Interpret`.
- Not in game (menu, loading, no server copy, broken chain) -> `Word` returns `ok=false`, so the Reader falls through to the save.
- Reads are cached for `MaxAge` (200 ms); the cached server address is revalidated on every refresh (type 0, class, id, extended stat list, PlayerData
  name) and rescanned when that fails or the pointer chain breaks. A State() other than InGame drops the address.
- Quest index and waypoint level->bit tables come from `internal/saves` (learned once with a scratch save, like `cheat.questWords`), not copied.
- Cost: a refresh is about 20 small reads; a scan about 50 ms, only at game start.

## Not verified (U)
- Done/clear/reward words after a real quest completion (only started/left-town bits and manual edits seen).
- Whether the quest words of an act the character never entered are in the buffer before the first visit (they were, for the edited words).
- Behaviour in a multi-player game (several server player units; the scan matches id, class and name of the local player only).
- Memory after the character dies/respawns, and across Save and Exit followed by "start" in the same process (the Source rescans on State change; the
  hand-built test covers it, live not run).

## Requests for INTEGRATE (not done: not my directories)
- Wire `questmem.New(reader)` first in the `quest.Reader` sources; call `Invalidate()` on game start if convenient (not required).
- `docs/progression/verification.md` V-05 and `docs/progression/results.md`: record the above (pointer chain, server-unit finding).
- Add `memquest i5` to `tools/instances.conf` (done in this branch).

## Wired into d2bot (INTEGRATE2, 2026-10-28, V live on i4 in `--managed` mode)
- `cmd/d2bot/task.go`: `saveQuestReader` asks `questmem.New(e.Read)` BEFORE the save (`quest.NewReader(mem, SaveSource)`); a quest the memory knows is reported with `task.EvMemory`, the
  save stays prior knowledge (`EvSave`) for everything memory cannot answer. Waypoints: memory wins where it knows the bit. The dialog row status (`status`) uses the same order.
  `TaskDeps.QuestsLive` is true then, so the task layer expects in-session changes. Every change of the Normal words is logged as `memory quest words` (plus `memory quest words unavailable ... why`).
- **Finding: a managed game is read through a plain file (`launch.MemReader`), which cannot list the game's private mappings, so `FindServerPlayer` failed with "source cannot list regions"**.
  `d2.RegionsOfPID` (from /proc/PID/maps) now backs a `RegionLister` on the bot's swappable memory source (`swapSource.Regions`). With it the memory words appeared and equalled the save (den_of_evil 0x9001,
  sisters_burial_grounds 0x0010 at the start). Right after an area change or a portal arrival the server unit is "not found" for a few seconds (the Source rescans); the reader then falls back to the save.
- Confirmation of a completion in the same session (Akara's claim, a boss kill) was NOT seen: the Den was already claimed on Booti, and no Blood Raven kill finished (see docs/boss.md section 6).
