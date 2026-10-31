# Memory layer: internal/mem, internal/d2, cmd/d2snap

## API summary (stable names)

`internal/mem`
- `Source` (`ReadAt(p []byte, addr uint32) error`, all-or-nothing), helpers `U8/U16/U32/I32`, `Base = 0x400000`.
- `Fake` (`Put`, `PutU32`), `Proc` (`FindGames`, `FindGame`, `Open(pid)`, `OpenGame()` = find + open + `VerifyBuild`, `Close`, `Alive`).
- `VerifyBuild(Source)`: PE32 at 0x400000, SizeOfImage 0x5ba000, TimeDateStamp 0x574ddfbc (works on a Snapshot too).
- Errors: `ErrNoGame`, `ErrUnmapped`, `ErrShort`, `ErrBuild`, `*ReadError{Addr,Len,Err}` (use `errors.Is`).
- `Recording` (`NewRecording(src)`, `Pages()`, `Snapshot()`), `Snapshot` (`NewSnapshot`, `Put`, `ReadAt`, `WriteTo`, `ReadSnapshot`, `LoadSnapshot`, `Save`).
  File format: `"D2SN"`, u16 version 1, gzip{u32 pageCount, pageCount x (u32 addr, 4096 bytes)}. A read of a missing page fails with `ErrUnmapped`.
  Envelope for all fixture kinds: see docs/replay.md (`OpenEnvelope`, `WriteEnvelope`, `SaveEnvelope`, `LoadSnapshotHeader`); legacy files load as schema 0.

`internal/d2` (`r := d2.New(src)`; `r.Retries`, `r.Sleep` tune torn-read retry)
- `r.State()` -> `StateMenu | StateInGame | StateLoading` (canary rule + full area chain + `pUpdateUnit == 0`).
- `r.Snapshot()` -> `*Snapshot{State, Player, Inventory, Monsters, Objects, Missiles, Ground, UI, Mouse, Screen, Viewport, Hovered}`.
  Not in game: only State/UI/Mouse/Screen set, err nil. Torn reads (player addr/id/area changed during the read, or a pointer chain broke) are retried, then reported as `StateLoading`. A real read failure (process gone) is returned as an error.
- `r.Player()` (`Player{Name, Class, X, Y, Mode, Act, Area, Life.., Level, Exp, Gold, StashGold, attributes, StatPoints, SkillPoints, Resist, Stats, Skills, LeftSkill, RightSkill, Difficulty}`), `Skill{ID, Level, Bonus}` with `Effective()`.
- `r.Units(type)` (`TypeMonster/Object/Missile/Item/Tile`) -> `[]Unit{Addr, Type, Class, ID, Mode, X, Y, Flags, Owner*, Monster *MonsterInfo, Item *Item}`; `Nearest`, `GroundItems`.
- `r.Inventory()` -> `Inventory{Items, Cursor}` with `Container(ContInventory|ContStash|ContCube|ContBelt|ContEquipment)` and `Equipped(Body*)`; `Item{Quality, Flags, ILvl, Body, Page, NodePage, Where, GX, GY}`.
- `r.UI()` (`UIState`, `Open(UIInventory...)`, `AnyPanelOpen()`), `r.Mouse()`, `r.ScreenSize()`, `r.Viewport()`, `WorldToScreen(wx, wy, vp)`, `r.Hovered()`, `r.AreaOf(unit)`, `r.ReadStatList`, `r.UnitStats`.
- Positions are subtiles. Life/mana/stamina are already divided by 256. Gold and other stats are raw.

`cmd/d2snap` (live, via `tools/live.sh mem d2snap ...`): `record=FILE[,key]` writes a fixture and replays it, plus exploration subcommands (`hex RVA:LEN`, `stats`, `inv`, `skills`, `player`, `mon`, `obj`, `ui=i,c,...`, `hover`, `sweep`, `cursor=X,Y`).
Fixtures: `internal/d2/testdata/*.snap` (see tests in `internal/d2/fixture_test.go`).

Tags: VERIFIED LIVE = checked on the running game (the worker host, 2026-10-28, character "Testy", level 1 Sorceress, Rogue Encampment). UNVERIFIED = not checked.

## Answers to docs section 7 (live, read-only)

### Q1 UI panel flags: VERIFIED LIVE
Method: read the bytes of GetUiVar_I (RVA 0x538D0) with `d2snap hex`, disassembled with objdump:
`cmp ecx,0x26; jb ok; ...assert...; ok: mov eax,[0x7a27c0+ecx*4]; ret`.
So the array is `DWORD ui[0x26]` at RVA 0x3A27C0 (VA 0x7a27c0). Then toggled panels with the d2input helper (`key press X`, wait 500 ms, read the array, press again) and diffed:
| key | index that flipped 0 -> 1 |
|---|---|
| i | 0x01 inventory |
| c | 0x02 character |
| t | 0x04 skill tree |
| q | 0x0F quest log |
| (Esc, seen when a restore loop pressed Esc with nothing open) | 0x09 ESC menu; while it is open 0x15, 0x22 and 0x23 read 0 |
At rest in town: [0], [0x15] (mini panel), [0x22], [0x23] are 1. `AnyPanelOpen` ignores those four. `p` (party) changed nothing offline; `h` toggles [0x21] (meaning unknown). [0x22] was 1 at first and read 0 after an `h` press and stayed 0 for the rest of the session (independent of the pointer position); meaning unknown, ignored by `AnyPanelOpen`. Other indexes in `d2/ui.go` come from D2BS and are UNVERIFIED.
Dead end / caution: pressing Esc with no panel open OPENS the ESC menu, so never use "Esc until the array equals baseline" blindly; check `ui[UIEscMenu]` first.

### Q2 hovered/selected unit: decoded VERIFIED LIVE, runtime behaviour UNVERIFIED
GetSelectedUnit (RVA 0x67A10) disassembly: `if [0x7a6a94]==0 return 0; edx=[0x7a6a8c]; ecx=[0x7a6a78]; esi=call 0x463990`, and 0x463990 is `GetServerUnit(ecx=id, edx=type)`: `edx = type*0x200 + 0x7a5e70; bucket = id & 0x7f`. So: active flag RVA 0x3A6A94, unit id RVA 0x3A6A78, unit type RVA 0x3A6A8C. The stored values read (type 1, id 9) = a real town NPC (Kashya), consistent. Hovering the town guard (the only unit on screen) and sweeping the pointer over the whole viewport on a 20 px grid never set the flag, so I could not observe it turning on. `d2.Hover` implements the decoded rule.

### Q3/Q9 screen and viewport (bonus)
ScreenSizeX/Y (0x31146C/0x311470) = 800 x 600 VERIFIED LIVE. MouseX/Y track `d2input move` exactly VERIFIED LIVE. ViewportX/Y (0x3A520C/0x3A5208) with absX=(wx-wy)*16, absY=(wx+wy)*8 put a standing player at screen (400, 276) VERIFIED LIVE (one standing position; while walking it is off by up to ~40 px because the camera lags).

### Q4 merged stats: VERIFIED LIVE
The helper behind GetUnitStat (0x225480 -> 0x624f60 in VA) does: `vec = (list->flags & 0x80000000) ? list+0x48 : list+0x24` then a lookup. The player and monster stat lists have flag 0x80000000. For those lists the effective vector is `{Stat* @+0x48, WORD count @+0x4C}`, NOT the base vector at +0x24/+0x28 (that one is the unit's own base stats only). Proof from the player's lists: base vector has 14 entries (attributes, life..., 67/68/69 = 100); the merged vector has 20 entries and adds the equipped staff's stats: 23/24 (damage), 72/73 (durability), stat 107 sub 36 = 1 (+1 Fire Bolt), 172, and stat 68 reads 110 instead of 100. So max life, resistances and skill bonuses must be read from the merged vector; `d2.ReadStatList` does that, `ReadBaseStatList` gives the old behaviour.
Scaling: life/mana/stamina (6-11) are 24.8 fixed point (10240 = 40). Gold (14/15) and exp (13) scaling UNVERIFIED because the character has none (stats absent); treated as raw (standard ItemStatCost valshift 0).
Skill levels: `Skill.Level` (+0x28 of the Skill struct) is base points only (Fire Bolt 0); `+Bonus` comes from stat 107 sub skill id. Class/tab bonus stats (83, 188) are not added.

### Q5 monster HP: partly VERIFIED LIVE
Town NPC stat lists are Ex lists too; stat 6 and 7 both read 0x8000 (128.0 in 24.8). That is the engine's 0..128 monster scale, so `LifePct = stat6*100/stat7`. A damaged monster was not available, so the ratio form is UNVERIFIED. `GetUnitHPPercent` (RVA 0x79080) is NOT a monster reader: decoded, it walks the party/roster list at 0x7bb5bc, so it only serves party members.

### Q6 difficulty: VERIFIED LIVE (location), Normal only
GetDifficulty (RVA 0x4DCD0) is `mov al,[0x7a060c]; ret`, i.e. a BYTE at RVA 0x3A060C. It reads 0 (Normal) now. 1 Nightmare and 2 Hell are the standard values, UNVERIFIED.

## Struct facts confirmed live (all offsets from the 1.14d client)
- UnitAny: type 0, class 4, id 0xC, mode 0x10, data 0x14, act 0x18, path 0x2C, stats 0x5C, inventory 0x60, pUpdateUnit 0x70, info 0xA8, flags 0xC4, next-in-bucket 0xE4. `+0x8C/+0x8E` hold a WORD copy of (x, y).
- Path: xPos WORD at +2, yPos WORD at +6 (players, monsters); ObjectPath and ItemPath: DWORD x,y at +0x0C/+0x10; Path.pRoom1 at +0x1C.
- PlayerData (+0x14 of the player): name at +0 ("Testy").
- Item: `ItemData` quality +0, flags +0x18 (0x10 identified, 0x20000 start item), ilvl +0x2C, body location +0x44, ItemLocation +0x45 (0xFF for belt/equipped, 0 inventory), NodePage +0x69 (1 storage, 2 belt, 3 equipment), next item in the inventory list +0x64. Item unit mode: 0 stored, 1 equipped, 2 belt. Grid cell/belt slot = ItemPath +0x0C/+0x10: belt potions (0,0)..(3,0), tomes in the inventory at (9,2) and (9,3) (so the tomes are 1x1). Stash (ItemLocation 4) and cube (3) UNVERIFIED.
- Inventory: +8 owner, +0xC first, +0x10 last, +0x20 cursor, +0x28 count.
- Info: +4 first skill, +8 left, +0xC right. Starting Sorceress: left = Attack (0), right = Fire Bolt (36).
- MonsterData flags/enchants/unique no (+0x16/+0x1C/+0x26): UNVERIFIED, town NPCs read all zero.

## Dead ends
- ItemTxt table from `GetItemText` (RVA 0x2335F0: count at VA 0x96ca58, rows at *0x96ca5c, stride 0x1A8): the D2BS field offsets for code/size do not apply to 1.14d (garbage at +0x40, +0xFC). Item sizes/codes are left to the data layer.
- Screenshots of the worker host desktop capture other people's windows; do not do that.
