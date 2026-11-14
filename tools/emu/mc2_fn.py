"""MERC2: print the function that contains each given hex address (start = last `push ebp; mov ebp, esp` before it), from
all.txt in the scratch. Usage: mc2_fn.py ADDR [ADDR..]  (prints at most 400 lines each)."""
import os
import sys

lines = [" ".join(l.split()) for l in open(os.path.join(os.environ["D2_SCRATCH"], "all.txt")).read().splitlines()]
addr = [int(l.split()[0], 16) for l in lines]
import bisect
for a in sys.argv[1:]:
    v = int(a, 16)
    i = bisect.bisect_right(addr, v) - 1
    j = i
    while j > 0 and not (lines[j].split(None, 1)[1].startswith("push ebp") and lines[j + 1].split(None, 1)[1].startswith("mov ebp, esp")):
        j -= 1
    k = j + 1
    while k < len(lines) and not (lines[k].split(None, 1)[1].startswith("ret") and (k + 1 >= len(lines) or lines[k + 1].split(None, 1)[1].startswith("int3") or lines[k + 1].split(None, 1)[1].startswith("push ebp"))):
        k += 1
    print("== function %x contains %s" % (addr[j], a))
    print("\n".join(lines[j:min(k + 1, j + 400)]))
