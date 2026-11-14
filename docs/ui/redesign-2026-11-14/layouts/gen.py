#!/usr/bin/env python3
"""Writes the watch.layout/1 preset drafts of this design and checks them (docs/ui/redesign-2026-11-14/D-layouts.md).

  python3 -I docs/ui/redesign-2026-11-14/layouts/gen.py     # from the repo root; rewrites the *.json next to it

Checks, the same rules tools/uilint applies to web/layouts/*.json (rule layout-manifest): schema, spans inside the
columns, no overlap among the shown entries of a breakpoint, unique order, and every id is a panel manifest id, an
alias, a compat card or a reserved id of web/shell.json, or one of the NEW ids of this design (which S1 adds to
"reserved" first). Stdlib only. A design aid, not product code.
"""
import json, os, sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", "..", "..", ".."))
WEB = os.path.join(ROOT, "internal", "watch", "web")
NEW = ["needs-you", "orders", "health", "progress", "zone-map", "log"]  # grid and side panels of this design
STATUS = ["statusline", "hero-meters", "needs-count"]                   # the status band of the new presets
BAND = {"video": "stage", "zone-map": "grid", "log": "grid"}


def known_ids():
    shell = json.load(open(os.path.join(WEB, "shell.json")))["layout"]
    ids = set(shell["compat"]) | set(shell["reserved"]) | set(shell["aliases"])
    bands = {}
    pdir = os.path.join(WEB, "panels")
    for d in sorted(os.listdir(pdir)):
        m = json.load(open(os.path.join(pdir, d, "panel.json")))
        ids.add(m["id"])
        bands[m["id"]] = m["band"]
    return ids, bands, shell["compat"]


# name -> rows of (id, col, row, colSpan, rowSpan) for the 12-column grid breakpoints (wide and laptop share them),
# then the reading order used by the flow breakpoints (tablet, phone), as (id, colSpan on a tablet).
PRESETS = {
    "simple": {
        "density": "comfortable",
        "grid": [("onboarding", 1, 1, 12, 3), ("video", 1, 4, 8, 14), ("needs-you", 9, 4, 4, 3), ("orders", 9, 7, 4, 6),
                 ("health", 9, 13, 4, 2), ("progress", 9, 15, 4, 3), ("journey", 1, 18, 12, 5)],
        "flow": [("onboarding", 12), ("video", 12), ("needs-you", 12), ("orders", 12), ("health", 6), ("progress", 6), ("journey", 12)],
    },
    "standard": {
        "density": "dense",
        "grid": [("onboarding", 1, 1, 12, 3), ("video", 1, 4, 6, 11), ("map", 7, 4, 3, 6), ("zone-map", 7, 10, 3, 12),
                 ("needs-you", 10, 4, 3, 3), ("orders", 10, 7, 3, 7), ("health", 10, 14, 3, 2), ("log", 10, 16, 3, 6),
                 ("progress", 1, 15, 6, 3), ("journey", 1, 18, 6, 4), ("cards", 1, 22, 3, 3), ("hero-vitals", 4, 22, 3, 4)],
        "flow": [("onboarding", 12), ("video", 12), ("needs-you", 12), ("orders", 6), ("zone-map", 6), ("health", 6), ("progress", 6),
                 ("map", 6), ("log", 6), ("journey", 12), ("cards", 6), ("hero-vitals", 6)],
    },
    "stream": {
        "density": "comfortable",
        "grid": [("video", 1, 1, 9, 16), ("map", 10, 1, 3, 6), ("progress", 10, 7, 3, 3), ("orders", 10, 10, 3, 4), ("log", 10, 14, 3, 3)],
        "flow": [("video", 12), ("progress", 12), ("orders", 12), ("map", 6), ("log", 6)],
    },
    "debug": {
        "density": "dense",
        "grid": [("onboarding", 1, 1, 12, 3), ("video", 1, 4, 6, 11), ("map", 7, 4, 6, 11), ("needs-you", 1, 15, 3, 3), ("orders", 4, 15, 3, 7),
                 ("zone-map", 7, 15, 3, 12), ("log", 10, 15, 3, 7), ("health", 1, 18, 3, 2), ("progress", 1, 20, 3, 3),
                 ("journey", 1, 27, 6, 5), ("hero-vitals", 7, 27, 3, 4), ("why", 10, 27, 3, 4), ("activity", 1, 32, 3, 4),
                 ("gear", 4, 32, 3, 4), ("tally", 7, 32, 3, 4), ("currency", 10, 32, 3, 2), ("now", 1, 36, 6, 6), ("orders-map", 7, 36, 6, 10),
                 ("queue", 1, 42, 3, 3), ("thoughts", 4, 42, 3, 4), ("journal", 1, 46, 6, 5), ("chronicle", 7, 46, 6, 5),
                 ("quests", 1, 51, 3, 5), ("cards", 4, 51, 3, 3), ("bestiary", 7, 51, 3, 5), ("atlas", 10, 51, 3, 4),
                 ("locked", 1, 56, 3, 4), ("rules", 4, 56, 6, 4), ("diary", 10, 56, 3, 4), ("alerts", 1, 60, 6, 5), ("spend", 7, 60, 6, 3)],
        "flow": None,  # the grid's reading order, full width
    },
}


def build(name, spec, ids, bands, compat):
    every = [i for i in sorted(ids) if i not in ("risk", "whyline", "hero-now", "voice", "hero-status", "hero-detail",
                                                 "hero-meters", "alerts-count", "journey-chip")]
    band = lambda i: BAND.get(i) or bands.get(i) or ("grid" if i in compat else "side")
    out = {"schema": "watch.layout/1", "cols": 12, "rowHeight": 48, "density": spec["density"], "theme": "dark",
           "status": STATUS, "breakpoints": {}}

    def entries(rows):
        shown = [r[0] for r in rows]
        es = [{"id": i, "band": band(i), "col": c, "row": r, "colSpan": w, "rowSpan": h, "collapsed": False, "order": n}
              for n, (i, c, r, w, h) in enumerate(rows)]
        row = max(e["row"] + e["rowSpan"] for e in es)
        for i in every:  # every other panel is named and switched off, so the shell does not pack it after the rest
            if i in shown:
                continue
            es.append({"id": i, "band": band(i), "col": 1, "row": row, "colSpan": 12, "rowSpan": 2, "collapsed": False,
                       "order": len(es), "hidden": True})
            row += 2
        return es

    grid = entries(spec["grid"])
    flow = spec["flow"] or [(i, 12) for (i, _, _, _, _) in spec["grid"]]
    span = {i: (c, r, w, h) for (i, c, r, w, h) in spec["grid"]}

    def flowed(tablet):
        rows, r = [], 1
        for i, w in flow:
            h = span[i][3]
            rows.append((i, 1, r, w if tablet else 12, h))
            r += h
        # two half-width neighbours share a row on a tablet
        if tablet:
            fixed, k = [], 0
            while k < len(rows):
                a = rows[k]
                if a[3] == 6 and k + 1 < len(rows) and rows[k + 1][3] == 6:
                    b = rows[k + 1]
                    hh = max(a[4], b[4])
                    fixed += [(a[0], 1, a[2], 6, hh), (b[0], 7, a[2], 6, hh)]
                    k += 2
                else:
                    fixed.append(a)
                    k += 1
            # close the gaps the pairing left
            r, packed, k = 1, [], 0
            while k < len(fixed):
                a = fixed[k]
                if a[3] == 6 and k + 1 < len(fixed) and fixed[k + 1][1] == 7:
                    b = fixed[k + 1]
                    packed += [(a[0], 1, r, 6, a[4]), (b[0], 7, r, 6, b[4])]
                    r += a[4]
                    k += 2
                else:
                    packed.append((a[0], 1, r, a[3], a[4]))
                    r += a[4]
                    k += 1
            rows = packed
        return entries(rows)

    out["breakpoints"] = {
        "phone": {"width": 390, "cols": 12, "panels": flowed(False)},
        "tablet": {"width": 768, "cols": 12, "panels": flowed(True)},
        "laptop": {"width": 1280, "cols": 12, "panels": grid},
        "wide": {"width": 1920, "cols": 12, "panels": grid},
    }
    return out


def check(name, lay, ids):
    bad = []
    if lay["schema"] != "watch.layout/1":
        bad.append("schema")
    for bp, b in lay["breakpoints"].items():
        seen, orders = [], set()
        for e in b["panels"]:
            if e["id"] not in ids and e["id"] not in NEW:
                bad.append(f"{bp}: unknown id {e['id']}")
            if e["col"] < 1 or e["col"] + e["colSpan"] - 1 > b["cols"] or e["row"] < 1 or e["rowSpan"] < 1:
                bad.append(f"{bp}: {e['id']} outside the columns")
            if e["order"] in orders:
                bad.append(f"{bp}: order {e['order']} twice")
            orders.add(e["order"])
            if e.get("hidden"):
                continue
            for o in seen:
                if (e["col"] < o["col"] + o["colSpan"] and o["col"] < e["col"] + e["colSpan"]
                        and e["row"] < o["row"] + o["rowSpan"] and o["row"] < e["row"] + e["rowSpan"]):
                    bad.append(f"{bp}: {e['id']} overlaps {o['id']}")
            seen.append(e)
    return [f"{name}: {b}" for b in bad]


def main():
    ids, bands, compat = known_ids()
    bad = []
    for name, spec in PRESETS.items():
        lay = build(name, spec, ids, bands, compat)
        bad += check(name, lay, ids)
        with open(os.path.join(HERE, name + ".json"), "w") as f:
            json.dump(lay, f, indent=1)
            f.write("\n")
    for b in bad:
        print("FAIL", b)
    print("layouts:", ", ".join(PRESETS), "-", len(bad), "finding(s)")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
