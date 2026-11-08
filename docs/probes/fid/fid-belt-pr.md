# fid-belt-pr (TASK-78, matrix row IT09): belt pickup lands in the belt first, d2d against game.exe

> status: finished 2026-11-08 by agent decomp5 (Sonnet). Result: **NO DEVIATION on current upstream main; ALREADY UPSTREAM** (the belt PR, evidence pack pr-belt-evidence-pack.md, is in upstream: `test_game --belt-pickup` and `--auto-belt` ship in its tree). 780,000 oracle cases, 0 differ. No PR candidate. Nothing posted, nothing pushed.

Tags: V run (game.exe's own FUN_0062bad0 / FUN_00628ba0 / FUN_0063c790 / FUN_0063c600 run natively by the emulator, NOTHING hooked, compared with d2d's real `rules::pick_up` / `rules::auto_belt` byte for byte), D (data tables), V read, I, U.

## 1. Claim

A potion (or any Beltable item) picked up from the ground lands in the belt when the belt has a matching column or a free box, in the box game.exe picks; the shift-click to the belt (C->S 0x63) picks the same box.

## 2. Oracle runs (upstream main 1453f853, built on the worker host)

Commands (worker host, the user's own game files in `D2_MPQ_DIR`, `D2_PATCH_INSTALLER` the 1.14d patch): `tools/emu/d5_belt_run.sh TREE BUILD OUT 100000` and `tools/emu/d5_autobelt_run.sh TREE BUILD OUT 100000`, which run, in the tree's `tools/emu`, `belt_pickup.py gen 100000 1 / game` and `auto_belt.py gen 100000 1 / game`, and `test_game --belt-pickup` / `--auto-belt` on the same cases, then `diff`.

| Differential | Cases | Differ | Tag |
|---|---|---|---|
| pickup, FUN_0063c600 against `rules::pick_up` (8 belts incl. none x bottom row {empty, hp1, mp1, rvs, isc}^4 x 29 picks, plus 100,000 random stacked belts) | 390,000 | **0** | V run |
| shift-click, FUN_0054d520's chain against `rules::auto_belt` (same generator shape) | 390,000 | **0** | V run |

Outcome counts (identical on both sides, `tools/emu/d5_belt_sum.py`): of about 13,450 cases per picked code, hp1 to hp5 land in the belt 9,936 to 10,082 times, mp1 to mp5 9,949 to 10,051, rvs 9,798, rvl 9,730, hpo 8,413; isc 2,859 (only onto a same-scroll column bottom), tsc 252, the other 1x1 consumables (vps, yps, wms, elx, mpo, rps, bpl, 0sc) 229 to 278 (same-code column only); key, rin, jew, amu, aqv, gcv 0 (not Beltable or not 1x1). The rest go to the inventory (belt column full or no matching bottom).

AC #1 (free belt slots: the potion lands in the belt): ticked with the table above (V run). AC #2 (belt sizes tie to the belt base): `belts.txt` of the 1.14d MPQ (D): belt 12, sash 8, default 4, girdle 16, light belt 8, heavy belt 12, uber belt 16 boxes (the file's `numboxes`); `data->belts[index].boxes` is read from the same file, and the oracle's 8 belt codes run with the real boxes on both sides with 0 differences, so a belt-size mismatch would have shown (V run, D). AC #3 (shim R04): see B1.

## 3. Deviations

None found. The earlier D (isc/tsc sent to the inventory instead of a same-scroll column, 3,111 cases of 390,000) is the belt PR's, merged upstream (git log: 89fc48b4 tooling; the fix is in upstream's `rules::pick_up`); today's run shows 0.

## 4. Bot findings (described, not changed)

B1 `docs/d2d-deviations.md` R04 ("lands in the pack, never in the belt") and the flag `NoAutoBelt` in `internal/town/sim/sim.go` still describe the pre-fix behaviour. On current upstream main d2d auto-belts exactly as game.exe (V run above), so the shim is obsolete **for a lab binary built from current main**. The lab binary in use is older (lab3 base e36922ea, lab6 e417d2b8) and may still lack the fix: removing the flag needs the lab-binary rebase (TASK-92) first. Keep R04 with the removal condition "lab binary contains upstream's rules::pick_up belt PR (check: `test_game --belt-pickup` mode exists in the lab tree)". Not changed here.

## 5. Drill recipe (after the lab rebase)

Lab binary with the lab verbs: `give` 4 hp1 potions to the ground via `debug`, an empty light belt equipped, walk over and click a potion; `dump items` must list it with `loc` belt and the box game.exe picks (bottom row, left box first). Repeat with an isc on the ground and a belt whose bottom box holds an isc (belt) and with a full tome (inventory).
