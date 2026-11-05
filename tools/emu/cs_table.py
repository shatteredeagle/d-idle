# SPDX-License-Identifier: GPL-3.0-or-later
"""COVSTATES: the States.txt table as the patched 1.14d stack holds it (data D), one line per state.

    tools/emu/run-on-worker.sh mpq-table.sh states > $D2_SCRATCH/states.txt     (first line `state id ...`)
    python3 -I tools/emu/cs_table.py states.txt [ids]                            (any python, no emulator)

Prints id, name, group, curse, aura, active, immed, curable, remhit, stat, setfunc, remfunc, skill, itemtype for
the listed ids (default all), then the column counts. The table is Blizzard data and is not committed.
"""
import sys

COLS = ("group", "curse", "aura", "active", "immed", "curable", "remhit", "restrict", "stat", "setfunc", "remfunc", "skill")


def load(path):
    lines = [ln.rstrip("\n").split("\t") for ln in open(path, encoding="latin-1") if ln.strip()]
    head = next(i for i, ln in enumerate(lines) if ln[0] == "state")
    names = lines[head]
    rows = []
    for ln in lines[head + 1:]:
        rec = dict(zip(names, ln))
        if rec.get("id", "").isdigit():
            rows.append(rec)
    return rows


def main():
    rows = load(sys.argv[1])
    ids = {int(a) for a in sys.argv[2:]}
    for r in rows:
        if ids and int(r["id"]) not in ids:
            continue
        print(r["id"], r["state"], *(f"{c}={r.get(c, '')}" for c in COLS if r.get(c, "")))
    print("states", len(rows))
    for c in ("curse", "aura", "active", "immed", "curable", "remhit", "group", "stat", "setfunc", "remfunc", "skill"):
        print(f"{c}: {sum(1 for r in rows if r.get(c, '') not in ('', '0'))} rows non-empty")


main()
