# navexit22: why seed 17 could not walk from the Blood Moor to the camp

Tags: V verified (how), S secondary, U unverified.

## Evidence
- V (gate log ms2): arrival in level 2 logged "2 adjacent entries"; the first walk to town at 12:48:02 logged "nav no exit known yet, exploring" (no "nav exit" line), then ErrExplored, then `no exit to that level: level 1 from 2` on every retry. So Adjacent held level 1, but `findGate` had no candidate. navexit21's `keepBorders` guess (border lost on a refetch) was wrong.
- V (own d2d slot on the worker host, `--seed 17 --no-save`, nice 19, recorded 2026-11-03, fixture `internal/d2dworld/testdata/seed17`): layout has the camp (level 1) at 1056,832 56x40 and the Blood Moor (2) at 1000,824 56x96, i.e. the camp lies EAST of the Blood Moor (the seed 17 map is transposed against the seed the d2dworld docs were verified on). `debug nearby` in the Blood Moor: `1 56 8`, `3 16 96`. The camp's grid is 285x205 subtiles; its column 280 (the margin) is all '#'.
- V (replay through the real d2dworld and nav code): Blood Moor contact toward level 1 is x=279, y=145..159 (15 cells), `BorderCells(1)` 15 cells, `borderGates(1)` empty.

## Root cause (V by TestSeed17BloodMoorHasTownGate: fails before, passes after)
`borderGates` drops a border cell when the cell across is known and not walkable in the neighbour's grid. It indexed the neighbour's grid with this level's coordinates. d2dworld's default (OriginNone, what d2denv uses) gives each level its own zero-based frame, so the Blood Moor cell (280, 145..159) was read as camp (280, 145..159): the wall margin. All 15 cells dropped, no gate, explore, ErrExplored, ErrNoExit. It worked on the seed of the earlier live checks only because the neighbour's grid happened to be open at the wrongly mapped cells. Seed dependent because the layout (which side the camp lies on, its offset) comes from the seed.

## Fix
Map the cell across into the neighbour's frame: `ax = c.X + dx - adj.Rect.X + nb.Rect.X` (same for y). `adj.Rect` is where d2dworld places the neighbour in our frame, `nb.Rect` is its own frame (identical frames under OriginTiles, so that mode is unchanged). Result for seed 17: one gate at (279,152), push (287,152); the camp's column 0 is open at rows 106..118 (the 13-cell camp gate).

## Not verified
- U: walking through that gate in the real game (no bot run; the exit probes only). Checklist: Gate 1 seed 17 re-run, expect no `no exit to that level: level 1 from 2`, a "nav exit ... to 1 ... border true" line and arrival in level 1.
- U: probe fidelity of the replay: the recorded `debug blocked` answers are per tile centre (x.5, y.5); the world asks at tile column + 0.1 (same tile), treated as the same.
- The throwaway slot used `debug goto 2` once to reach the Blood Moor (read-only afterwards, `--no-save`, slot 2, stopped; no ledger entry: not the cheat package).
