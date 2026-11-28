#!/usr/bin/env python3
"""Split a one-file dark ledger into per-knob design files and the runtime state file (card led1). Stdlib only.

  migrate.py split [--legacy F] [--out DIR] [--state F] [--keep]   write DIR/<knob>.json (design), merge the runtime fields
                                                                   into the state file, remove the legacy file
  migrate.py state --legacy F [--state F]                          only import the runtime fields of F into the state file
                                                                   (seed the head box from the checkout the watcher wrote to)
  migrate.py verify --legacy F [--out DIR] [--state F]             read-only: the merged view equals the rows of F

`--state ""` skips the state file. The state default is the flip tool's own file (never a checkout).
"""
import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import ledger as L  # noqa: E402


def read_legacy(path):
    with open(path) as f:
        return json.load(f)["items"]


def import_state(items, state_path):
    """The runtime fields of every row replace that row's entry in the state file; other rows stay."""
    st = L.load_state(state_path)
    for it in items:
        _, rt = L.split_row(it)
        st["rows"][it["name"]] = rt
    L.save_state(state_path, st)
    return st


def split(legacy, out, state_path, keep=False):
    items = read_legacy(legacy)
    names = [it["name"] for it in items]
    if len(set(names)) != len(names):
        raise ValueError("duplicate row names in the legacy ledger")
    for it in items:
        L.write_row(out, L.split_row(it)[0])
    if state_path:
        import_state(items, state_path)
    if not keep:
        os.remove(legacy)
    return len(items)


def diff(legacy, out, state_path):
    """-> list of differences between the legacy rows and the merged view (empty: equal)."""
    want = {it["name"]: it for it in read_legacy(legacy)}
    got = {it["name"]: it for it in L.rows(out, state_path or None)}
    probs = [f"{n}: only in {'legacy' if n in want else 'split'}" for n in sorted(set(want) ^ set(got))]
    for n in sorted(set(want) & set(got)):
        for k in sorted(set(want[n]) | set(got[n])):
            if want[n].get(k) != got[n].get(k):
                probs.append(f"{n}.{k}: legacy {want[n].get(k)!r} split {got[n].get(k)!r}")
    return probs


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("cmd", choices=["split", "state", "verify"])
    ap.add_argument("--legacy", default=L.LEGACY)
    ap.add_argument("--out", default=L.DESIGN_DIR)
    ap.add_argument("--state", default=None)
    ap.add_argument("--keep", action="store_true", help="split: leave the legacy file in place")
    a = ap.parse_args(argv)
    state = L.default_state() if a.state is None else a.state
    if a.cmd == "split":
        n = split(a.legacy, a.out, state, a.keep)
        print(f"split {n} rows into {a.out}; runtime state in {state or '(skipped)'}")
    elif a.cmd == "state":
        if not state:
            ap.error("state needs a state file")
        st = import_state(read_legacy(a.legacy), state)
        print(f"imported runtime fields of {len(read_legacy(a.legacy))} rows into {state} ({len(st['rows'])} rows there)")
    else:
        probs = diff(a.legacy, a.out, state)
        for p in probs:
            print("DIFF", p)
        print(f"verify: {len(probs)} difference(s)")
        return 1 if probs else 0
    return 0


if __name__ == "__main__":
    sys.exit(main())
