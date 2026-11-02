#!/usr/bin/env python3
"""verifyaidel: read monstats (stdin, tab separated, patched stack) and summarize the AiDel columns.
Usage: tools/emu/run-on-worker.sh mpq-table.sh monstats > t.txt ; python3 -I tools/emu/va_aidel.py < t.txt
(or pipe the wrapper output through this locally; the first line must start with 'Id\\thcIdx')."""
import sys, collections
lines = [l.rstrip("\n").split("\t") for l in sys.stdin if "\t" in l]
i = next(k for k, l in enumerate(lines) if l[0] == "Id" and l[1] == "hcIdx")
h = lines[i]
rows = [r for r in lines[i + 1:] if len(r) == len(h)]
cols = [k for k, n in enumerate(h) if n.lower().startswith("aidel")]
print("rows", len(rows), "aidel cols", [h[k] for k in cols])
for k in cols:
    c = collections.Counter(r[k] for r in rows)
    print(h[k], dict(sorted(c.items(), key=lambda x: -x[1])[:12]))
    z = sum(1 for r in rows if r[k] in ("", "0"))
    print("  empty-or-0:", z)
# per-row difference between difficulties
if len(cols) == 3:
    diff = [(r[0], [r[k] for k in cols]) for r in rows if len({r[k] for k in cols}) > 1]
    print("rows with differing N/NM/H:", len(diff))
    for d in diff[:15]:
        print("  ", d)
st = [k for k, n in enumerate(h) if n.lower() in ("enabled",)]
en = [r for r in rows if st and r[st[0]] == "1"]
print("enabled rows", len(en))
for k in cols:
    print(h[k], "enabled:", dict(collections.Counter(r[k] for r in en)))
