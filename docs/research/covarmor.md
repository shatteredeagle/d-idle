# Armor bases and item types: first-pass trace (covarmor)

Status: first pass, data only. No game.exe code was run and no d2d source was read in this job, so nothing here is `V run` or `V read`. No ledger rows are claimed.

Tags: D = derived from the patched MPQ tables (`tools/emu/run-on-worker.sh mpq-table.sh armor` and `itemtypes`, patch layer `Patch_D2.mpq`), U = unverified, I = inferred.

## Q1. What does armor.txt hold? (D)
- 202 data rows plus one separator row with blank cells (the tool prints 203 lines). 202 matches `docs/coverage/denominators.tsv` items.armor.
- Defence: `minac`/`maxac`. 0 rows have minac > maxac. Examples: Buckler 4..6, Quilted Armor 8..11, Tower Shield 22..25, Gothic Plate 128..135, Sacred Targe 126..158.
- Required strength `reqstr`: 0 to 232. Examples: Buckler 12, Tower Shield 75, Gothic Plate 70.
- `block` (base block chance, shields only): 149 rows 0; non-zero values 3 to 30 on the shield types. Tower Shield 24, Sacred Targe 30.
- `speed` (armor speed penalty): 0 on 166 rows, 5 on 24 rows, 10 on 12 rows (Gothic Plate 5, Tower Shield 10).
- `durability`: one row has 0; `nodurability` is set on 0 rows.
- `gemsockets` (max sockets): 0 on 45 rows, 1 on 2, 2 on 52, 3 on 64, 4 on 39.
- Item class chain: `normcode`/`ubercode`/`ultracode` give the normal, exceptional and elite version. 69 rows are their own normal base, 68 their own exceptional and 68 their own elite (Buckler buc/xuc/uuc). Sacred Targe (`pab`) is its own elite and points back to `pa1` normal.
- `type` values: tors 45, helm 24, shie 24, ashd 15, belt 15, boot 15, glov 15, head 15, pelt 15, phlm 15, circ 4. `type2` is blank on every row.

## Q2. What does itemtypes.txt hold for armor? (D)
- 103 data rows plus one blank row (104 lines; denominator says 98 codes, the 5 extra are blank-cell or duplicate-code rows: U, not checked).
- Equip slots (`BodyLoc1`/`BodyLoc2`): `rarm`/`larm` 30 rows (shields and weapons), head 4, tors 2, belt, feet, glov, neck once each, rings `rrin`/`lrin`.
- Equiv chains: shie -> shld; tors, boot, glov, belt -> armo; helm -> (none); head (voodoo heads) -> shld -> necr; ashd (auric shields) -> shld -> pala; phlm (primal helm) -> helm -> barb; pelt -> helm -> drui; circ -> helm. So a class-restricted item is both an armor type, a shield or helm type, and a class item.
- `Repair` flag: 32 rows 1, 71 rows 0. `Body` 1 on 42 rows (can be worn). `Normal`/`Magic`/`Rare` flags control which quality may spawn: Normal 31 rows, Magic 9, Rare 58.

## Q3. How do these feed defence, strength checks, speed and block? (U)
Not decomposed. Next steps in order:
1. game.exe: find the armor record loader (reads the columns above, blank = 0) and the defence roll that picks a value in `minac..maxac` for a generated item (V read), then run it for all 202 rows x quality (normal, superior, ethereal) with a fixed seed against d2d (V run).
2. Ethereal defence is believed to be base * 1.5 (rounded down) and superior +enhanced defence percent (I, from general game knowledge, not read in the binary here). A counted differential is needed before any claim.
3. Required-strength equip check: game.exe compares character strength with `reqstr` (minus any item stat reduction); the same for ethereal items (-10, I). Check the exact ordering with a grid of strength 0..300 per row.
4. Block: shield block = base `block` per class, modified by dexterity and level (U); compare d2d.
5. `speed` column is a penalty on the faster-hit-recovery/run chain (U); locate in game.exe and in d2d.

## Deviation list
None proven. Candidates (all U, differential buildable from the `tools/emu/*_run.sh` templates by running the item defence generation with table rows as input, oracle = d2d result for the same seed):
- D1? Blank cells as 0 versus a d2d default (rows: block, speed, reqstr, durability).
- D2? Zero-durability row (one row has durability 0 without `nodurability`): game.exe treats durability 0 and `nodurability` differently from d2d?
- D3? Ethereal and superior defence ordering/rounding.
- D4? `type2` blank everywhere: d2d may still read it; check.

## Bot findings
None. The bot's equip and requirement handling was not read in this pass (U).

## What could not be read
d2d sources, the fork checkout and the game.exe disassembly were not opened in this job.

## PR candidate
No PR candidate: no deviation proven by a differential.

## Matrix-cell updates
None. No ledger rows added (no case run). Suggested rows once the checks above run: `items.armor` and `itemtypes` with evidence this doc.

## Shared-file lines for the merger
See `openspec/changes/covarmor/merge-lines.txt`.
