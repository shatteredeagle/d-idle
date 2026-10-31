# internal/data: static game tables

Package `d2lod-bot/internal/data` (tables, cache, lookups) and `d2lod-bot/internal/data/mpq` (MPQ reader).
Standard library only. Nothing Blizzard owns is committed: the tables are extracted at setup time from the
user's own install into a local cache directory. Tests use only hand-written excerpts, plus the real files when
present (they skip cleanly when `/tmp/d2mpq` is absent).

## API (what the bot, nav and combat code imports)

```go
// Setup / startup
data.Extract(installDir, cacheDir string) error      // MPQs -> compact cache (stamp.json written last)
data.CacheCurrent(installDir, cacheDir string) bool  // cache exists and matches the install
data.Load(cacheDir string) (*DB, error)              // read the cache (about 35 ms)
data.Open(installDir, cacheDir string) (*DB, error)  // re-Extract if stale, then Load: the normal entry point
data.LoadInstall(installDir string) (*DB, error)     // straight from MPQs, no cache (about 90 ms)

// Lookups on *DB (all return nil / zero for unknown ids; DB is read-only and goroutine-safe)
db.MonsterByClassID(id) *Monster        // id = unit txtFileNo = monstats hcIdx
db.MonsterByName("skeleton1"|"Skeleton")*Monster
db.SuperUniqueByID(id) / SuperUniqueByName(name) *SuperUnique
db.AreaByID(id) / AreaByName("Blood Moor") *Level
db.SkillByID(id) / SkillByName("Fire Bolt") *Skill      // skill.ManaCost(lvl), skill.Synergies()
db.ClassByName("Sorceress") *CharClass
db.ExpToLevel(level) int64              // total exp needed to reach level (1..99); db.Exp.LevelFor(exp)
db.ItemByCode("hax") *ItemBase          // weapons+armor+misc by 3-4 char code (runes are misc r01..r33)
db.ItemType("swor") *ItemType; db.IsA("swor","weap") bool
db.ObjectByID(id) *Object
db.InventoryLayout("Sorceress"|"Bank Page 1"|"Transmogrify Box Page 1"|"Hireling") *Inventory  // grid + slot rects
db.BeltLayout("belt") / BeltByIndex(i) *Belt
db.TreasureClassByName(n) / UniqueByName / SetItemByName / RunewordByName
db.MonLevelRow(mlvl), db.ScaledHP(m, diff, mlvl), db.Difficulties[diff] (resist penalty, death exp penalty)
db.Str(key) / db.Text(key)              // string tables; Text strips colour codes
data.Town(act) / data.IsTown(areaID)
```

Also exported on `DB` as plain slices: `Monsters, SuperUniques, Levels, Skills, Classes, ItemTypes, Items, Objects,
Inventories, Belts, Treasure, Uniques, SetItems, Runewords, Cube, MonLevels`. Difficulty arrays are indexed by
`data.Normal/Nightmare/Hell`; resistances by `data.Physical/Magic/Fire/Lightning/Cold/Poison`
(`m.Immune(data.Fire, data.Hell)`; 100 or more is immune, negative is a weakness).

Generic layer: `data.ParseTable(bytes)` gives a `Table` (header row, case-insensitive column lookup, `Row.Str/Int/Float/Bool`
taking one or more alias names; missing columns and blank or non-numeric cells read as zero values) and
`data.ParseTBL(bytes)` parses `.tbl` string files. `mpq.Open(path)` / `mpq.OpenReaderAt` give an `*mpq.Archive`
with `ReadFile(path)`, `Has`, `Size`, `ListFile`.

## Cache

`Extract` writes one trimmed TSV per table (only the columns listed below, same row order and count, header row kept),
`strings.tsv` (key, tab, Go-quoted value; the three .tbl files merged), and `stamp.json`. Total about 1.2 MB for 1.14d.
The stamp is `{schema, sizes of Game.exe, Patch_D2.mpq, d2exp.mpq, d2data.mpq}`; `CacheCurrent`/`Open` compare it with
the install (sizes, not mtimes, so copying the install does not invalidate it). Bump `schemaVersion` in `schema.go`
whenever a column list changes so existing caches are rebuilt. The stamp is written last, so an interrupted
extraction is never loaded. The loaders read the same column names from the cache and from raw tables, so a column
that is missing in some version just reads as zero.

## Where the data lives (1.14.3.71 install on the worker host)

Install dir: `~/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II/`. File names are matched
case-insensitively (the install has `Patch_D2.mpq`). Priority order, highest first, same as the game's own:

1. `Patch_D2.mpq` (5.7 MB, 210 blocks, no `(listfile)`): the 1.14 patch. It carries the current `.txt` tables (see below)
   and `data\local\lng\eng\patchstring.tbl`. Plain `.txt` files, not just `.bin`.
2. `d2exp.mpq` (250 MB): LoD. Holds both `.bin` and `.txt` for every table in `data\global\excel\`, plus
   `expansionstring.tbl` and an older `patchstring.tbl` (35 KB, shadowed by the patch's 53 KB one).
3. `d2data.mpq` (269 MB): base game tables (`.txt` + `.bin`) and `string.tbl`.

Not needed and not copied: `d2char`, music/sfx/speech/video archives. Paths inside the archives use `\`
(`data\global\excel\monstats.txt`, `data\local\lng\eng\string.tbl`); the reader accepts `/` too and is case-insensitive.
The language directory is hard-coded to `eng`.

Which archive each table actually came from in this install: everything in the list below came from `Patch_D2.mpq`
except `belts` (d2exp), and the string tables `string.tbl` (d2data), `expansionstring.tbl` (d2exp),
`patchstring.tbl` (Patch_D2). The loader still falls through the full priority list per table, so installs where the
patch lacks a table work. Merge order for strings: `patchstring` over `expansionstring` over `string`.
d2data/d2exp copies of the same tables differ from the patch (e.g. `monstats.txt` is 238 KB in d2data, 393 KB in d2exp,
433 KB in the patch), so priority order matters; the cache must use the patch version.

## MPQ reader notes

- Header v1 (`MPQ\x1a`, 32 bytes) found at offset 0 (scans 512-byte boundaries, honours `MPQ\x1b` user-data headers);
  v2+ (64-bit extended tables) is rejected. All three archives are v1 with sector size 4096.
- Hash/block tables decrypted with the standard crypt table and keys `(hash table)` / `(block table)`. Lookup is by
  path hash (name A/B hashes, linear probing), no listfile needed. `(listfile)` exists in d2data and d2exp and is used only by tests.
- Compression masks seen when decoding every file listed in d2data+d2exp: `0x08` PKWARE implode (about 14.6k sectors:
  nearly everything, including all `.txt`/`.tbl`), `0x02` zlib (about 600), `0x41` Huffman+ADPCM mono (about 240) and
  `0x81` Huffman+ADPCM stereo (4): the last two are `.wav` audio only. Audio is not decoded: those 572 files in d2exp
  return an "unsupported compression mask" error, everything else (d2data: 0 failures, d2exp: only the .wav) decodes.
  No file uses bzip2, sparse, or patch-file blocks in these archives, but bzip2 (stdlib) is implemented and tested on a synthetic archive.
- Multi-compression masks are undone in the order bzip2, implode, zlib. Only single-method masks occur in the real data
  and are tested; the combined-order guess follows StormLib's behaviour and is UNVERIFIED.
- Sector files: offset table of n+1 entries (+1 when the sector-CRC flag is set; CRCs are ignored). A sector whose stored
  size equals its expected size is raw (no mask byte). Single-unit files, encrypted files (file key from the base name,
  optional FIX_KEY adjustment, per-sector key + index) are implemented and tested on synthetic archives; real D2 tables
  are not encrypted.
- PKWARE DCL "explode" is a Go port of the algorithm in zlib's `contrib/blast` (tables copied from there). Tested
  against synthetic streams (literal-only, coded literals, matches, end code) and by successfully extracting every real
  table.
- `mpq.Build` writes tiny single-unit archives for tests; it is not on the extraction path.

## 1.14d data versus older docs

Verified by reading the real files (not live in game):

- `monstats.txt`: the class id is the explicit **`hcIdx`** column, not the row number. There is an `Expansion` separator
  row (hcIdx blank) that is not a class and must not count. 734 classes (ids 0..733, with gaps removed by
  the separator only). Display name via `NameStr` -> string tables.
- `levels.txt`: area id is the `Id` column (137 areas, 0..136), `Act` is 0-based, an `Expansion` separator row exists.
  Monster levels per difficulty for LoD are `MonLvl1Ex/2Ex/3Ex` (the plain `MonLvl1..3` are classic); the loader prefers `Ex`.
  `LevelName` is the English display name directly in the table.
- `experience.txt`: first data row is `MaxLvl` (99), then rows `0..99`. Row `n` holds the total exp needed to reach level `n+1`
  (row 1 = 500 = level 2, row 98 = 3,520,485,254 = level 99, row 99 = cap). The `ExpRatio` column is the per-level multiplier (1024 = full).
- `skills.txt`: `Id` column is the skill id; class codes in `charclass` are three letters (`ama sor nec pal bar dru ass`).
  Synergies are not columns: they are expressions in `EDmgSymPerCalc`/`DmgSymPerCalc` such as
  `(skill('Fire Ball'.blvl)+skill('Meteor'.blvl))*par8`; `Skill.Synergies()` extracts the referenced skills.
  Mana: `(mana + lvlmana*(lvl-1)) << manashift / 256`, minimum `minmana` (Fire Bolt level 1 = 5<<7/256 = 2.5). The formula
  reproduces the commonly quoted Fire Bolt cost; UNVERIFIED against the in-game tooltip.
  Skill names are not in skills.txt: `skilldesc` -> `skilldesc.txt` column `str name` -> string table.
- Resist columns in monstats are `ResDm ResMa ResFi ResLi ResCo ResPo`, with `(N)`/`(H)` variants (so Physical is "Dm").
  Blank means 0. Andariel (class 156): poison 80/50/66, fire -50 on all difficulties (read from the file).
- `monlvl.txt` has two column sets: plain (classic) and `L-` (LoD). The loader prefers `L-`. `ScaledHP` multiplies
  monstats MaxHP by the `L-HP` percentage; for Andariel it gives roughly 1.0k / 24.8k / 60.0k (N/NM/H) which is in the
  range community sources quote, UNVERIFIED against a live monster. Bosses and uniques add their own multipliers.
- `difficultylevels.txt`: resist penalty 0/-40/-100, death exp penalty 0/5/10.
- `inventory.txt`: every panel has two rows, e.g. `Sorceress` and `Sorceress2`; the first set has the inventory panel at
  x 320..640 (640x480 layout), the `2` set at x 400..720 (800x600 layout, a 80 px / 60 px shift). Which one the running
  1.14d client at 800x600 uses is UNVERIFIED; the 800x600 set is the likely match for our windowed 800x600 game.
  Grid is 10x4 cells of 29 px for characters, 6x8 for the LoD stash ("Big Bank Page 1"; classic "Bank Page 1" is 6x4),
  3x4 Horadric Cube ("Transmogrify Box Page 1"). Which stash row the 1.14d LoD client uses is UNVERIFIED (Big Bank is the 6x8 one). Slot rects use -1 for "no such slot" and are dropped.
- `belts.txt`: rows `belt(12) sash(8) default(4) girdle(16) light belt(8) heavy belt(12) uber belt(16)`, an `Expansion` row, then the
  same set with a `2` suffix. A belt item's `armor.txt` `belt` column is the row index; `misc.txt` `autobelt` marks potions that
  go to the belt.
- `itemtypes.txt`: types inherit through `Equiv1/Equiv2` (`swor` -> `mele` -> `weap`); `DB.IsA` walks that.
- `runes.txt` is the runeword table: `Name` is an internal id (`Runeword130`), `Rune Name` is the string-table key
  ("Spirit"), rune columns `Rune1..6` are misc.txt codes (`r07 r10 r09 r11`).
- String tables are UTF-8 with colour codes encoded as U+00FF `c` digit (`ÿc1`); `CleanString` removes them.
  Item names use the 3-4 char item code as the key (`hax` -> "Hand Axe"); unique/set names use the `index` text itself as key.
- Older docs often describe the d2s/classic data with different column names (`MonLvl1` instead of `MonLvl1Ex`, plain monlvl
  columns, no `hcIdx`, no `Expansion` separator handling). The loaders accept the alias where one is trivial (`MonLvl*Ex`
  then `MonLvl*`, `L-` then plain monlvl) but only 1.14d was actually read.

## Tables we use (columns kept in the cache, and why)

| Table | Used for | Columns |
|---|---|---|
| monstats | immunities/resists per difficulty, levels, boss/minion/undead/demon flags, HP/AC/exp/damage per difficulty, treasure classes, group sizes | Id hcIdx BaseId NameStr MonType AI enabled rangedtype minion1/2 SetBoss PartyMin/Max MinGrp/MaxGrp Velocity Run Level(+N,H) isMelee npc interact lUndead hUndead demon flying boss primeevil killable Res*(+N,H) ToBlock(+N,H) minHP/maxHP AC Exp A1/A2/S1 min,max,TH (+N,H) El1-3 Mode/Type/Pct/MinD/MaxD TreasureClass1-4(+N,H) |
| levels | area ids, act, parent, exits/neighbours (Vis0-7, Warp0-7), monster levels and density per difficulty, spawn lists, waypoint, display name | Name Id Act Layer SizeX/Y(+N,H) OffsetX/Y Depend Teleport IsInside DrlgType LevelType Vis0-7 Warp0-7 Portal Position Quest MonLvl1-3(+Ex) MonDen(+N,H) MonUMin/Max(+N,H) NumMon mon/nmon/umon 1-10 Waypoint LevelName LevelWarp |
| skills | mana cost, level/stat requirements, prerequisites, range, damage (physical and elemental, with per-level steps), synergy expressions, cooldown, passive/aura/summon flags, params | skill Id charclass skilldesc reqlevel maxlvl reqstr/dex/int/vit reqskill1-3 range passive aura periodic InTown summon delay leftskill srvmissile mana lvlmana manashift minmana SrcDam HitShift Min/MaxDam(+Lev1-5) DmgSymPerCalc EType EMin/EMax(+Lev1-5) EDmgSymPerCalc ELen ELevLen1-3 calc1-4 Param1-8 |
| skilldesc | skill display name and skill-tree position (page/row/column) for allocating points | skilldesc SkillPage SkillRow SkillColumn str name |
| charstats | class base stats, per-level gains, velocities, block factor, starting skill and items | class str dex int vit stamina hpadd LifePerLevel StaminaPerLevel ManaPerLevel LifePerVitality StaminaPerVitality ManaPerMagic StatPerLevel ToHitFactor Walk/RunVelocity RunDrain BlockFactor StartSkill Skill 1-10 item1-10 (+loc,count) |
| experience | exp thresholds per level, death penalty basis | Level Amazon ExpRatio (all class columns are identical in vanilla) |
| itemtypes | type hierarchy, body locations, ammo/launcher links, belt/charm/gem flags, socket limits | ItemType Code Equiv1/2 Body BodyLoc1/2 Shoots Quiver Throwable AutoStack Charm Gem Beltable MaxSock1/25/40 Class StorePage |
| weapons, armor, misc | item size (grid cells), stackability and max stack, requirements, durability, damage/AC, belt placement, quality upgrade codes (normal/exceptional/elite), potion/scroll effect | name code namestr type type2 invwidth invheight stackable minstack maxstack level levelreq reqstr reqdex cost durability nodurability gemsockets speed rarity spawnable quest useable throwable belt autobelt normcode ubercode ultracode mindam maxdam 2handmindam 2handmaxdam 2handed wclass minac maxac block pSpell state len stat1 calc1 |
| objects | what a clickable object is (waypoint, shrine, chest, door), size, whether selectable, how it operates | Name Id Token SizeX SizeY Selectable0 IsDoor BlocksVis SubClass Lockable Act OperateRange ShrineFunction Restore AutoMap Left Top Width Height OpenWarp Parm0-7 |
| superuniques | named bosses/uniques (Bishibosh ... Nihlathak), their base monster, modifiers and treasure class | Superunique Name Class hcIdx Mod1-3 MinGrp MaxGrp AutoPos Stacks TC(+N,H) |
| inventory | pixel layout of the inventory grid, stash, cube, equipment slots: used to turn (col,row) into a click position | class inv* grid* <slot>Left/Right/Top/Bottom/Width/Height for rArm torso lArm head neck rHand lHand belt feet gloves |
| belts | potion belt slot rectangles per belt type | name numboxes boxwidth boxheight box1-16 left/right/top/bottom |
| treasureclassex | what a monster/chest can drop and with what weights (loot filter, farming value) | Treasure Class group level Picks Unique Set Rare Magic NoDrop Item1-10 Prob1-10 |
| uniqueitems | unique names, base code, level, properties (loot evaluation) | index version enabled ladder rarity lvl lvl req code prop/par/min/max 1-12 |
| setitems | set item names, base, properties | index set item rarity lvl lvl req prop/par/min/max 1-9 (partial-set aprop* skipped) |
| runes | runeword recipes: runes in order, allowed base types, properties | Name Rune Name complete server itype1-6 etype1-3 Rune1-6 T1Code/Param/Min/Max 1-7 |
| cubemain | Horadric Cube recipes (inputs/outputs only, mods not loaded) | description enabled ladder min diff op param value class numinputs input 1-7 output output b output c |
| monlvl | per-monster-level multipliers for AC, to-hit, HP, damage, exp (danger/exp estimates) | Level L-AC/TH/HP/DM/XP(+N,H) and the plain set |
| difficultylevels | resist penalty and death exp penalty per difficulty | Name ResistPenalty DeathExpPenalty |
| string.tbl, expansionstring.tbl, patchstring.tbl | key -> display name for monsters, skills, items, uniques, sets, runewords | (all keys, merged) |

Not extracted (add to `schema.go` plus a loader if needed later): itemstatcost, properties, states, missiles, magicprefix/suffix,
gems, sets (full set bonuses), hireling, lvlmaze/lvlprest/lvlsub/lvlwarp (level generation presets; the map reader gets real geometry from memory),
shrines, monpreset, pettype, npc.

## Dead ends and caveats

- No Huffman/ADPCM/sparse support: only audio uses them here.
- `Level.Waypoint` is the raw column (255 = none); `Object.SubClass` is a raw bit field (waypoints show 64). Their meaning is
  kept raw on purpose; decode only after checking in game.
- `Skill` damage fields are raw numbers/expressions; the combat agent decides how far to evaluate them
  (damage scaling is `min + (lvl-1)*step` style per-level fields plus synergy percentages).
- Class ids: `CharClass.ID` is the row order in charstats (Amazon 0 .. Assassin 6), matching the game's class numbering
  (UNVERIFIED against live memory; the MEM agent reads the real value).
- Tests that read real MPQs: `go test ./internal/data/...` (needs `/tmp/d2mpq` with d2data.mpq, d2exp.mpq, Patch_D2.mpq and Game.exe;
  copied from the worker host with scp). `-short` skips the slow full-archive decode in the mpq package.
