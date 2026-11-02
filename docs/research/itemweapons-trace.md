# Weapon and item-type bases: first-pass trace (itemweapons)

Status: first pass, data only. No game.exe code was run for this doc, so nothing here is `V run`. No ledger rows are claimed.

Tags: D = derived from the patched MPQ table (`tools/emu/run-on-worker.sh mpq-table.sh weapons`, the patch layer `Patch_D2.mpq`), U = unverified.

## Q1. What does weapons.txt hold? (D)
- 306 rows (matches `docs/coverage/denominators.tsv` items.weapons). Weapon classes (`wclass`): 1hs 122, stf 56, 1ht 44, bow 30, 2ht 21, ht1 21, xbw 12.
- Damage columns: `mindam`/`maxdam` (one-hand) and `2handmindam`/`2handmaxdam` (two-hand). 0 rows have mindam > maxdam. Two-hand-only bases (`2handed`=1): 137 rows, they carry only the 2hand pair (example: Giant Thresher 7wc 40..114).
- `speed` (weapon speed modifier, the WSM): distinct values -60, -40, -30, -20, -15, -10, -5, blank (0), 5, 10, 20. Examples: Hand Axe and Short Sword blank (0); Katar -10; Phase Blade -30; Giant Thresher -10.
- Strength/dexterity bonus percent: 300 of 306 rows have `StrBonus` or `DexBonus` set. Hand Axe and Short Sword 100 str; Katar 75/75 (the claw class `ht1`). Others 100 str only.
- Requirements `reqstr`/`reqdex`: blank means 0 (Hand Axe, Short Sword); Phase Blade 25/136, Giant Thresher 188/140.
- Durability: `durability` 0 on Phase Blade with `nodurability` set on 49 rows (U: whether d2d treats 0 as indestructible the same way).

## Q2. How do these feed melee damage and attack speed? (U)
Not decomposed yet. Next steps, in order:
1. Find in game.exe the table loader that fills the weapon record from weapons.txt (columns above) and confirm blank = 0 and the speed sign convention (V read first).
2. Run the damage formula for a counted set (for example all 306 rows x strength/dexterity grid, fixed seed) under the emulator and compare against d2d's weapon rules byte for byte (V run).
3. Attack-speed: WSM enters the frame-count formula with the class base frames; run game.exe for each `wclass` against d2d.

## Deviation list
None proven. Candidate checks (all U, differential could be built from `tools/emu/*_run.sh` templates by running the item stat/damage code with table rows as input):
- D1? Blank cells read as 0 versus d2d default (rows: speed, reqstr, reqdex, durability).
- D2? Two-hand damage choice for `1or2handed` rows (the barbarian/one-hand rule).

## Bot findings
None. The bot's own weapon handling was not read in this pass (U).

## What could not be read
d2d sources and the fork checkout are not readable in this job; game.exe was not disassembled here.

## Next scope
itemtypes.txt (98 rows: equiv chains, equip slots), armor.txt (202 rows: defence, block, reqstr, durability), then the formulas above.

## PR candidate
No PR candidate: no deviation proven by a differential.

## Matrix-cell updates
None (no matrix doc covers item bases yet).

## Shared-file lines for the merger
See `openspec/changes/itemweapons/merge-lines.txt`.
