#!/usr/bin/env python3
"""Fidelity risk map: rank where d2d may differ from game.exe in ways that move our gated metrics.

Reads the upstream gap data (gaps.json, functions.tsv; fetch with `riskmap.py fetch DIR`)
and our dependency list (dependencies.json), prints the table ranked by impact x (1 - confidence).
Only counts, module names, addresses and statuses are used; no upstream code is read or stored.
Stdlib only. Run with `python3 -I tools/riskmap/riskmap.py ...`.
"""
import argparse
import csv
import json
import os
import re
import sys
import urllib.request

BASE = "https://psi29a.github.io/d2decomped/"
FILES = ("gaps.json", "functions.tsv")
STATUSES = ("verified", "ported", "cited", "untouched")
HERE = os.path.dirname(os.path.abspath(__file__))


def fetch(dest):
    os.makedirs(dest, exist_ok=True)
    for name in FILES:
        with urllib.request.urlopen(BASE + name, timeout=60) as r, open(os.path.join(dest, name), "wb") as f:
            f.write(r.read())
    print("fetched", ", ".join(FILES), "to", dest)


def load(data_dir):
    with open(os.path.join(data_dir, "gaps.json"), encoding="utf-8") as f:
        gaps = json.load(f)
    with open(os.path.join(data_dir, "functions.tsv"), encoding="utf-8", newline="") as f:
        funcs = list(csv.DictReader(f, delimiter="\t"))
    return gaps, funcs


def count(rows):
    c = dict.fromkeys(STATUSES, 0)
    for r in rows:
        if r["status"] in c:
            c[r["status"]] += 1
    return c


def analyse(area, gaps, funcs):
    """Return the data columns for one dependency area."""
    files = set(area.get("d2d_files", []))
    ref_rows = {}
    for r in funcs:
        paths = set(re.findall(r"((?:components|apps)/[\w/]+\.\w+):\d+", r.get("refs", "")))
        if paths & files:
            ref_rows[r["address"]] = r
    exe_files = {x.lower() for x in area.get("exe_files", [])}
    exe_mods = set(area.get("exe_modules", []))
    exe_rows = [r for r in funcs
                if r["library"] != "true" and r["dead"] != "true"
                and (r["file"].lower() in exe_files or r["module"] in exe_mods)]
    by_addr = {r["address"]: r for r in funcs}
    addrs = {a: by_addr[a]["status"] if a in by_addr else "absent" for a in area.get("addresses", [])}
    srcs = area.get("gap_src", [])
    listed = [g for g in gaps["gaps"] if any(g["source"].endswith("/" + s) or g["source"] == s for s in srcs)]
    ponytails = sum(1 for g in gaps["gaps"] if g["kind"] == "ponytail" and g["section"] in files)
    score = area["impact"] * (1 - area["confidence"])
    return {
        "id": area["id"], "name": area["name"], "impact": area["impact"],
        "confidence": area["confidence"], "score": round(score, 2),
        "d2d": count(ref_rows.values()), "exe": count(exe_rows),
        "addresses": addrs, "ponytails_in_files": ponytails,
        "listed_gaps": [{"src": g["source"], "kind": g["kind"], "title": g["title"][:80]} for g in listed],
        "cost_h": area["cost_h"], "covered_by": area["covered_by"], "first_step": area["first_step"],
    }


def rank(rows):
    return sorted(rows, key=lambda r: (-r["score"], r["cost_h"], r["id"]))


def fmt(c):
    return "%d/%d/%d/%d" % tuple(c[s] for s in STATUSES)


def table(rows):
    out = ["| # | area | impact | conf | risk | d2d refs v/p/c/u | game.exe v/p/c/u | listed gaps | cost h | covered by |",
           "|---|---|---|---|---|---|---|---|---|---|"]
    for i, r in enumerate(rows, 1):
        out.append("| %d | %s | %d | %.2f | %.2f | %s | %s | %d | %d | %s |" % (
            i, r["id"], r["impact"], r["confidence"], r["score"], fmt(r["d2d"]), fmt(r["exe"]),
            len(r["listed_gaps"]), r["cost_h"], r["covered_by"]))
    return "\n".join(out)


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("cmd", nargs="?", default="rank", choices=["rank", "fetch", "detail"])
    p.add_argument("dir", nargs="?", default=os.path.join(HERE, "data"), help="data directory (default tools/riskmap/data)")
    p.add_argument("--deps", default=os.path.join(HERE, "dependencies.json"))
    p.add_argument("--json", action="store_true")
    p.add_argument("--top", type=int, default=0)
    a = p.parse_args(argv)
    if a.cmd == "fetch":
        fetch(a.dir)
        return 0
    gaps, funcs = load(a.dir)
    with open(a.deps, encoding="utf-8") as f:
        deps = json.load(f)
    rows = rank([analyse(x, gaps, funcs) for x in deps["areas"]])
    if a.top:
        rows = rows[:a.top]
    g = gaps["generated"]
    if a.json:
        json.dump({"commit": g["commit"], "date": g["date"], "areas": rows}, sys.stdout, indent=1)
        print()
        return 0
    print("source: psi29a/d2decomped commit %s, %s; risk = impact x (1 - confidence)" % (g["commit"], g["date"]))
    print(table(rows))
    if a.cmd == "detail":
        for r in rows:
            print("\n## %s: %s" % (r["id"], r["name"]))
            print("addresses:", ", ".join("%s=%s" % kv for kv in r["addresses"].items()) or "-")
            for lg in r["listed_gaps"]:
                print("  %s %s: %s" % (lg["kind"], lg["src"], lg["title"]))
    return 0


if __name__ == "__main__":
    sys.exit(main())
