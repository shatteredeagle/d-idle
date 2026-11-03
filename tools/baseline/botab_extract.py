#!/usr/bin/env python3
"""BOTAB extractor: bot-level metrics per run and per clvl band from committed minute-line files.

Read-only, stdlib only. Definitions (same for both arms, d2dsoak conventions):
  hours   = minute lines / 60 (virtual game hours; restart gaps excluded)
  xp      = net character XP (later minus earlier line), interval attributed to the clvl of the earlier line
  kills, deaths, potions = sum of positive increments of the running counters (they reset at a bot restart)
  town trips = entries into the town area (area 1, 40, 75, 103, 109: act town ids) from a non-town line
  time per area = share of minute lines per area id
Usage: botab_extract.py NAME=PATH[.jsonl|.jsonl.gz] ... > out.json
"""
import gzip
import json
import sys

BANDS = [("1-4", 1, 4), ("5-7", 5, 7), ("8-10", 8, 10), ("11-14", 11, 14), ("15-20", 15, 99)]
TOWNS = {1, 40, 75, 103, 109}


def load(path):
    op = gzip.open if path.endswith(".gz") else open
    out = []
    with op(path, "rt") as f:
        for line in f:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                d = json.loads(line)
            except ValueError:
                continue
            if d.get("msg") == "minute":
                out.append(d)
    return out


def band_of(c):
    for n, lo, hi in BANDS:
        if lo <= c <= hi:
            return n
    return "?"


def inc(a, b, key):
    d = b.get(key, 0) - a.get(key, 0)
    return d if d > 0 else 0


def pot(m):
    return m["potionsUsed"] if "potionsUsed" in m else m.get("potions", 0)


def extract(ms):
    res = {}
    for i in range(1, len(ms)):
        a, b = ms[i - 1], ms[i]
        r = res.setdefault(band_of(a["clvl"]), {"minutes": 0, "xp": 0, "kills": 0, "deaths": 0, "potions": 0,
                                                  "town_trips": 0, "restarts": 0, "areas": {}, "town_minutes": 0})
        r["minutes"] += 1
        r["xp"] += b["xp"] - a["xp"]
        r["kills"] += inc(a, b, "kills")
        r["deaths"] += inc(a, b, "deaths")
        r["potions"] += max(0, pot(b) - pot(a))
        r["restarts"] += inc(a, b, "restarts")
        if b["area"] in TOWNS and a["area"] not in TOWNS:
            r["town_trips"] += 1
        if a["area"] in TOWNS:
            r["town_minutes"] += 1
        r["areas"][str(a["area"])] = r["areas"].get(str(a["area"]), 0) + 1
    return res


def main():
    global BANDS
    out = {}
    for arg in sys.argv[1:]:
        if arg.startswith("--bands="):  # e.g. --bands=12-13,11-14 (each line goes to the first band that holds it)
            BANDS = [(b, int(b.split("-")[0]), int(b.split("-")[1])) for b in arg[8:].split(",")]
            continue
        name, path = arg.split("=", 1)
        ms = load(path)
        whole = {"minutes": len(ms) - 1, "first_clvl": ms[0]["clvl"], "last_clvl": ms[-1]["clvl"]}
        out[name] = {"whole": whole, "bands": extract(ms)}
    json.dump(out, sys.stdout, indent=1, sort_keys=True)


if __name__ == "__main__":
    main()
