#!/usr/bin/env python3
"""verifynmlevel: the rule on a full grid, from game.exe's own tables and FUN_0061dca0.

    labrun <worker> -- bash docs/research/pr-nmlevel-verifier/py.sh vnl_grid.py OUTFILE

For every MonStats row (game.exe's loaded .bin, [0x744304]+0xa78, 0x1a8 a row), every difficulty and every level
id 0..140 (0, the towns and ids past Levels.txt's last included: edges), the start level FUN_00573cb0 computes in
an expansion game (0x573e15..0x573e53, read in the disassembly): the row's Level column (+0xaa + 2 x diff) unless
diff > 0 and the row byte +0xc has neither mask bit ([0x6ce270] = 4, [0x6ce280] = 0x40), in which case FUN_0061dca0
(level id, diff, expansion 1), which runs natively here. One line each:
  G cls diff lid col flags start
Also checks that byte +0xc bit 2 / bit 6 are MonStats.txt's noRatio / boss columns for every row (stderr).
"""
import sys

import drlg
import monstats


def main():
    e = drlg.boot()
    tables = e.r32(0x744304)
    rows, base = e.r32(tables + 0xa80), e.r32(tables + 0xa78)
    mask_noratio, mask_boss = e.r32(0x6ce270), e.r32(0x6ce280)
    txt = [r for r in monstats.table("monstats") if r.get("id") and r["id"] != "Expansion"]
    bad_cols = 0
    with open(sys.argv[1], "w") as out:
        for cls in range(rows):
            row = base + cls * 0x1a8
            flags = e.read(row + 0xc, 1)[0]
            if cls < len(txt):
                want = (4 if txt[cls].get("noratio") == "1" else 0) | (0x40 if txt[cls].get("boss") == "1" else 0)
                if flags & 0x44 != want:
                    bad_cols += 1
                    print(f"row {cls} {txt[cls]['id']}: byte +0xc {flags:#x}, txt noRatio/boss {want:#x}", file=sys.stderr)
            for diff in range(3):
                col = e.r16(row + 0xaa + 2 * diff)
                col = col - 0x10000 if col >= 0x8000 else col
                for lid in range(0, 141):
                    start = col
                    if diff > 0 and not flags & mask_noratio and not flags & mask_boss:
                        start = e.call(0x61dca0, lid, diff, 1)
                        start = start - (1 << 32) if start >= 1 << 31 else start
                    out.write(f"G {cls} {diff} {lid} {col} {flags} {start}\n")
    print(f"{rows} rows; noRatio/boss byte vs MonStats.txt mismatches: {bad_cols}; txt rows {len(txt)}", file=sys.stderr)


if __name__ == "__main__":
    main()
