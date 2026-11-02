# SPDX-License-Identifier: GPL-3.0-or-later
"""PALDUR: which game.exe code can reach FUN_0057d4f0 (the durability / on-hit post-processor) by DIRECT calls and jumps.

    tools/emu/run-on-worker.sh vq_dis.py --all $D2_SCRATCH/all.txt        # the linear disassembly (disasm.py --all)
    tools/emu/run-on-worker.sh pal_callgraph.py > out.txt                  # reads $D2_SCRATCH/all.txt

Function starts: an address after 0xCC padding (int3 run) or `push ebp; mov ebp, esp`; a `call` / `jmp` to an absolute address inside
the image is an edge (a jmp to another function start is a tail call). Indirect calls (tables of function pointers: skill do /
start functions, missile hit functions) are NOT followed, so a "no" is a statement about direct edges only (I, not V).
Prints: the direct callers of the roots, the closure of callers upward from FUN_0057d4f0 (names of the known entries), and for
each known missile-side entry whether FUN_0057d4f0 / FUN_0057d3d0 / FUN_00559e30 is reachable downward.
"""
import os
import re
import sys
from collections import defaultdict

PATH = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "all.txt")
line_re = re.compile(r"^([0-9a-f]{8})\s+(\S+)\s*(.*)$")

addrs, ops = [], {}
with open(PATH) as f:
    for line in f:
        m = line_re.match(line)
        if not m:
            continue
        a = int(m.group(1), 16)
        addrs.append(a)
        ops[a] = (m.group(2), m.group(3).strip())

starts = set()
prev_pad = True
for i, a in enumerate(addrs):
    mn, arg = ops[a]
    if mn == "int3":
        prev_pad = True
        continue
    if prev_pad:
        starts.add(a)
        prev_pad = False
    if mn == "push" and arg == "ebp" and i + 1 < len(addrs) and ops[addrs[i + 1]] == ("mov", "ebp, esp"):
        starts.add(a)
sorted_starts = sorted(starts)

import bisect  # noqa: E402


def func_of(a):
    k = bisect.bisect_right(sorted_starts, a) - 1
    return sorted_starts[k] if k >= 0 else None


calls = defaultdict(set)      # function start -> set of function starts it calls / tail-jumps to
callers = defaultdict(set)
for a in addrs:
    mn, arg = ops[a]
    if mn in ("call", "jmp") and arg.startswith("0x"):
        try:
            t = int(arg, 16)
        except ValueError:
            continue
        if t not in ops:
            continue
        src = func_of(a)
        if src is None:
            continue
        tgt = func_of(t)
        if mn == "jmp" and tgt == src:
            continue
        if mn == "call" or t in starts:
            calls[src].add(t if t in starts else tgt)
            callers[t if t in starts else tgt].add(src)

TARGETS = {0x57D4F0: "FUN_0057d4f0", 0x57D3D0: "FUN_0057d3d0", 0x559E30: "FUN_00559e30"}
print("direct callers of FUN_0057d4f0:", " ".join(f"{c:x}" for c in sorted(callers[0x57D4F0])))
print("direct callers of FUN_0057d3d0:", " ".join(f"{c:x}" for c in sorted(callers[0x57D3D0])))
print("direct callers of FUN_00559e30:", " ".join(f"{c:x}" for c in sorted(callers[0x559E30])))

# upward closure from FUN_0057d4f0
seen, todo = {0x57D4F0}, [0x57D4F0]
while todo:
    x = todo.pop()
    for c in callers[x]:
        if c not in seen:
            seen.add(c)
            todo.append(c)
print(f"functions with a direct-call path to FUN_0057d4f0: {len(seen)}")

ENTRIES = {
    0x5ADF10: "missile hits a unit (FUN_005adf10)",
    0x5AD8A8: "missile hit builder region (0x5ad8a8, mid-function: its start is used)",
    0x5A89A0: "missile physical roll (FUN_005a89a0)",
    0x57DBF0: "apply a hit record (FUN_0057dbf0)",
    0x57C6C0: "on-hit events (FUN_0057c6c0)",
    0x5A7670: "monster attack handler (FUN_005a7670)",
    0x56F070: "player plain Attack do 1 (FUN_0056f070)",
    0x5A5490: "monster melee blow (FUN_005a5490)",
    0x5A6D50: "monster fires a missile (FUN_005a6d50)",
}


def reach(root):
    seen, todo = {root}, [root]
    while todo:
        x = todo.pop()
        for c in calls[x]:
            if c not in seen:
                seen.add(c)
                todo.append(c)
    return seen


for addr, name in ENTRIES.items():
    root = func_of(addr)
    r = reach(root)
    hits = [n for t, n in TARGETS.items() if t in r]
    print(f"{addr:x} -> function {root:x} {name}: {len(r)} functions reachable; reaches {', '.join(hits) if hits else 'none of the three'}")
