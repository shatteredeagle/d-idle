#!/usr/bin/env python3
"""alignment2 reading aid: context and containing function from the linear sweep ($D2_SCRATCH/all.txt).

    al2_ctx.py ctx ADDR [BEFORE AFTER]     # lines around ADDR
    al2_ctx.py func ADDR                   # the function (push ebp; mov ebp,esp ... next such pair) containing ADDR
    al2_ctx.py callers ADDR                # every `call 0xADDR`, with the function start that holds it
    al2_ctx.py imm TEXT                    # every line whose text contains TEXT (e.g. "push 0xac"), with function start
"""
import os
import sys

OUT = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "all.txt")
lines = open(OUT).read().splitlines()
idx = {ln[:8]: i for i, ln in enumerate(lines)}


def fstart(i):
    s = i
    while s > 0 and not (lines[s].split(None, 1)[1:] == ["push ebp"] and lines[s + 1].endswith("mov ebp, esp")):
        s -= 1
    return lines[s][:8]


def main():
    cmd = sys.argv[1]
    if cmd == "batch":
        return
    if cmd == "ctx":
        i = idx[sys.argv[2].zfill(8)]
        b = int(sys.argv[3]) if len(sys.argv) > 3 else 12
        a = int(sys.argv[4]) if len(sys.argv) > 4 else 6
        print("\n".join(lines[i - b:i + a + 1]))
    elif cmd == "func":
        i = idx[sys.argv[2].zfill(8)]
        s = i
        while not (lines[s].split(None, 1)[1:] == ["push ebp"] and lines[s + 1].endswith("mov ebp, esp")):
            s -= 1
        e = i + 1
        while not (lines[e].split(None, 1)[1:] == ["push ebp"] and lines[e + 1].endswith("mov ebp, esp")):
            e += 1
        print("\n".join(lines[s:e]))
    elif cmd == "callers":
        t = "call    0x" + sys.argv[2].lstrip("0x")
        t2 = "call 0x" + sys.argv[2].lstrip("0x")
        for i, ln in enumerate(lines):
            if ln.endswith(t) or ln.endswith(t2):
                print(ln[:8], "in", fstart(i))
    elif cmd == "imm":
        for i, ln in enumerate(lines):
            if sys.argv[2] in ln:
                print(ln, "in", fstart(i))


def batch(specs):
    """each spec: ctx:ADDR:B:A | func:ADDR | callers:ADDR (one process, several dumps)"""
    for sp in specs:
        print("=====", sp)
        sys.argv = ["al2_ctx.py", *sp.split(":")]
        main()


if sys.argv[1] == "batch":
    batch(sys.argv[2:])
else:
    main()
