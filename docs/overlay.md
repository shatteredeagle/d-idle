# Table overlay (d.idle mode): file format and rules

> status: implemented, 2026-11-06, TASK-59. Fork side `--overlay-dir` on branch `d2d.lab4-overlay` of the private fork (not yet pinned in any gate); bot side `internal/overlay`, `internal/data` and `d2bot --overlay-dir`. Design and reasons: [research/custom-items.md](research/custom-items.md), [research/metacurrency-design.md](research/metacurrency-design.md).
> Tags: **V** verified (how is named); **U** unverified.

Vanilla mode is the default and untouched. d.idle mode adds OUR OWN rows on top of the user's own game tables at run time. A fragment never contains Blizzard table text: only rows we wrote, and for a replaced row only the key plus the cells we change.

## Layout

```
<overlay dir>/excel/<Table>.txt     one fragment per table, tab separated
```

The file name is the table name (case-insensitive: `Misc.txt`, `UniqueItems.txt`, `TreasureClassEx.txt`). Strings (`.tbl`) and art are NOT overlaid yet (U: see "Not done").

## Fragment format

- First line: column names, exactly as in the base table header (case-insensitive). Any subset, any order, but the key column must be present. An unknown column is an error.
- Following lines: one row each, the same number of tab separated cells as the header. LF or CRLF. Blank lines are skipped.
- A row whose key is already in the base table REPLACES that row's cells where the fragment cell is non-empty (an empty cell keeps the base value).
- Any other row is APPENDED; columns the fragment does not name are empty. Appended rows get ids after the vanilla rows (affix and unique ids are row numbers), so existing saves keep their meaning.
- Errors (fatal at startup, no silent fallback): unknown column, wrong cell count, missing key column, empty key, a key twice in one fragment, empty file, a fragment for a table the install lacks. The message is `overlay <File>.txt:<line>: <why>` (1-based line; V by `tests/test_overlay.cpp` in the fork and `internal/overlay` tests).

## Key per table

| table | key column | on a key match |
|---|---|---|
| Misc, Armor, Weapons, Gems | `code` | replace |
| ItemTypes | `Code` | replace |
| Properties | `code` | replace |
| UniqueItems, SetItems, Sets | `index` | replace |
| TreasureClassEx | `Treasure Class` | replace |
| Runes | `*Rune Name` | replace |
| ItemStatCost | `Stat` | replace |
| SuperUniques | `Superunique` | replace |
| MonStats | `Id` | replace |
| Skills | `skill` | replace |
| CubeMain, MagicPrefix, MagicSuffix, AutoMagic | none (append only) | always append: these rows have no unique name (affix names repeat) |
| any other table | first column | replace |

The table is in both implementations (fork `key_column`, Go `keyColumn`); change both. If the base has the key twice, the first row is the one replaced.

Prefer NEW private rows over replacing: a custom unique gets an `index` with our prefix; a drop route is a new private treasure class plus one pointer cell changed in an existing class (a diff, not a copy of the row).

## Examples (FAKE rows, ours)

`excel/Misc.txt`, a stackable token on an existing key-like base:

```
code	name	type	namestr	invwidth	invheight	stackable	maxstack
zzm1	Fake Mark	key	zzm1	1	1	1	12
```

`excel/TreasureClassEx.txt`, a private class and one changed cell of an existing class (the existing class named `Some Class` is replaced only in `Item3`/`Prob3`):

```
Treasure Class	Picks	NoDrop	Item1	Prob1
DI Fake Mark	1	0	zzm1	1
```

## Mode stamp and hash

- Hash (identical in the fork and in `internal/overlay`; V pinned by a test on both sides with the value `7af2328b` for one fragment `A.txt` holding `X`): FNV-1a 64 over, per fragment sorted by lower-case file name, the lower-case name, a 0 byte, the file bytes, a 0 byte; the stamp is the first 8 hex digits of the 16.
- d2d control channel `info` ends its line with ` overlay=<hash8>` or ` overlay=off`.
- Bot mode stamp: `overlay.Mode(dir)` is `vanilla` or `didle:<hash8>` (for saves, run records, probes; used by TASK-62/64).

## Switches

| side | switch | default |
|---|---|---|
| d2d fork | `--overlay-dir <dir>` | off; without it the table bytes are untouched (V by reading the code path: `Stack::try_read` returns the archive bytes unless an overlay was set) |
| d2bot | `--overlay-dir <dir>` or env `D2_OVERLAY_DIR`; needs `--backend d2d`; passed to a `--managed` d2d only when set | off |
| internal/data | env `D2_OVERLAY_DIR` | off |

## Go extract cache

`data.Stamp` has `overlay` (the hash8; omitted in vanilla, so the vanilla stamp JSON and the `CacheDir` hash of existing caches are unchanged). Fragments are merged into the raw table before the column trim, so a cache from vanilla is never current in d.idle mode and the reverse. `data.Open` puts a d.idle cache in `<cacheDir>/didle-<hash8>/`; `OpenCached` gets a separate directory from the stamp hash. Tests: `internal/data/overlay_test.go`.

## Not done (U)

- Strings (`.tbl`) and art: a new row shows its key until the string path is built; the first slice relies on the key text.
- Whether game.exe or d2d tolerates a d.idle save in vanilla mode (policy: refuse by offline scan, TASK-62).
- Vanilla table-hash equality (overlay off vs the unchanged build) and the full fork ctest suite were not run (V only: test_overlay and overlay_d2d pass on the worker host).

## First shipped overlay: didle-v1 (TASK-63, 2026-11-06)

`data/overlay/didle-v1/excel/` holds the first real fragments: the Raven's Mark currency (`mrk`), one treasure-class line and one cube bank row. Rows, rates and checks: [research/currency-rows.md](research/currency-rows.md); probe record: [probes/currency1.md](probes/currency1.md). Offline validation (`internal/overlay/didlev1_test.go`) needs the user's own install and skips without it.

Donor columns. A new Misc row needs the numbers that make a valid row of the Skeleton Key kind, which are game constants and not text. They are copied as VALUES only, with the minimum set: `type`, `alternategfx`, `component`, `invwidth`, `invheight`, `flippyfile`, `invfile`, `dropsound`, `dropsfxframe`, `usesound`, `transtbl` (what the art and sounds are called). Ours: `name`, `code`, `namestr`, `level`, `levelreq`, `rarity`, `spawnable`, `cost`, `nodurability`, `stackable`, `minstack`, `maxstack`, `spawnstack`. The replaced Blood Raven treasure row carries only the key and the two cells `Item5`, `Prob5`.
