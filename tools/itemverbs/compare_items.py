#!/usr/bin/env python3
"""Compare two `items` dumps (the d2d devctl verb) of the same scenario, one done by the click path and one by the item
verbs (docs/design/item-verbs.md section 9, plan item 3). Unit ids differ between runs, so items are compared by code,
location, slot, panel and cell, and gold/scroll counts by the extra `state` line. Usage:
  compare_items.py click.items verb.items
Each file is the raw reply of `items` (and optionally a `state` line starting with `state:` copied in). Exit 0 = same."""
import re, sys

def rows(path):
    text = open(path).read()
    out = sorted(
        (m.group(1), int(m.group(2)), int(m.group(3)), int(m.group(4)), int(m.group(5)), int(m.group(6)))
        for m in re.finditer(r"\[(\w+) loc=(\d+) slot=(\d+) q=\d+ panel=(\d+) id=-?\d+ size=\d+x\d+ at=(\d+),(\d+)\]", text))
    gold = re.findall(r"gold=(\d+)", text)
    return out, gold

if len(sys.argv) != 3:
    sys.exit(__doc__)
a, b = rows(sys.argv[1]), rows(sys.argv[2])
if a == b:
    print(f"same: {len(a[0])} items, gold {a[1]}")
    sys.exit(0)
for name, x, y in (("only in click run", a[0], b[0]), ("only in verb run", b[0], a[0])):
    extra = [r for r in x if r not in y]
    if extra:
        print(name + ":", *extra, sep="\n  ")
if a[1] != b[1]:
    print("gold differs:", a[1], b[1])
sys.exit(1)
