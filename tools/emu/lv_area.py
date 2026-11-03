# SPDX-License-Identifier: GPL-3.0-or-later
"""game.exe's area monster level FUN_0061dca0(levelId, difficulty, expansion) run natively for every Levels.txt id,
against the Levels.txt columns d2d keeps in GameData::area_level (MonLvl1Ex..3Ex) and the classic MonLvl1..3
(agent levelsareas).

    tools/emu/run-on-worker.sh lv_area.py > out.txt

Output: `A lid diff exp game table` per case, then a summary line `cases N mismatches M` (stdout).
"""
import drlg
import monstats


def main():
    e = drlg.boot()
    rows = {int(r["id"]): r for r in monstats.table("levels") if r.get("id", "").isdigit()}
    n = bad = 0
    for lid in range(0, 141):
        r = rows.get(lid)
        for expansion in (1, 0):
            for diff in range(0, 4):
                got = e.call(0x61dca0, lid, diff, expansion)
                got = got - (1 << 32) if got >= 1 << 31 else got
                if r is None:
                    want = None
                elif diff >= 3:
                    want = 1
                else:
                    key = f"monlvl{diff + 1}ex" if expansion else f"monlvl{diff + 1}"
                    v = r.get(key, "")
                    want = int(v) if v.lstrip("-").isdigit() else 0
                n += 1
                ok = want is None or got == want
                bad += 0 if ok else 1
                print("A", lid, diff, expansion, got, want, "" if ok else "DIFF")
    print("cases", n, "mismatches", bad)


if __name__ == "__main__":
    main()
