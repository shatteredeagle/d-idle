#!/usr/bin/env python3
"""Counts pointer call sites (Click, MoveMouse, ClickWorld, Key, Hold, Release, MoveTo on an input controller) in
behaviour packages, outside the adapters (input layer I6, docs/design/input-i5-i7.md, ADR 0021).

The count may fall, never rise: it fails when a package's count is above tools/ci/pointercount-baseline.json. Lower the
baseline with --update after a migration. The scan is a text match on receivers named In/in/inp/ctrl (a heuristic, S);
it is a ratchet toward 0 on the d2d path, not proof."""
import json, os, re, sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
BASE = os.path.join(ROOT, "tools", "ci", "pointercount-baseline.json")
ADAPTERS = ("internal/input/", "internal/d2dinput/", "internal/d2dctl/", "internal/d2dtown/")
PAT = re.compile(r"\b(?:In|in|inp|ctrl)\.(?:Click|MoveMouse|ClickWorld|Key|Hold|Release|MoveTo)\(")


def count(root=ROOT):
    per = {}
    for top in ("internal",):
        for d, _, files in os.walk(os.path.join(root, top)):
            rel = os.path.relpath(d, root).replace(os.sep, "/") + "/"
            if rel.startswith(ADAPTERS):
                continue
            for f in files:
                if not f.endswith(".go") or f.endswith("_test.go"):
                    continue
                with open(os.path.join(d, f), encoding="utf-8") as fh:
                    n = sum(1 for line in fh if PAT.search(line) and not line.lstrip().startswith("//"))
                if n:
                    pkg = rel.rstrip("/")
                    per[pkg] = per.get(pkg, 0) + n
    return per


def main(argv):
    per = count()
    total = sum(per.values())
    if "--update" in argv:
        with open(BASE, "w") as fh:
            json.dump({"total": total, "packages": dict(sorted(per.items()))}, fh, indent=1)
            fh.write("\n")
        print("pointercount: baseline written, total", total)
        return 0
    try:
        base = json.load(open(BASE))
    except OSError:
        print("pointercount: no baseline (run with --update)")
        return 1
    bad = [(p, n, base["packages"].get(p, 0)) for p, n in per.items() if n > base["packages"].get(p, 0)]
    for p, n, b in bad:
        print(f"pointercount: {p} has {n} pointer call sites, baseline {b}")
    print(f"pointercount: total {total} (baseline {base['total']}, target 0 on the d2d path)")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
