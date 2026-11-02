#!/usr/bin/env python3
"""Call-site finder for the SKSTATEEV decomposition (reading aid; output to stdout).

    sk_stateev_xref.py all            # build $D2_SCRATCH/all.txt (linear sweep of game.exe)
    sk_stateev_xref.py ADDR...        # every `call 0xADDR` with the 6 instructions before it
"""
import os
import sys

import disasm  # upstream tools/emu/disasm.py

OUT = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "all.txt")

if __name__ == "__main__":
    if sys.argv[1:] == ["all"]:
        sys.argv = ["disasm.py", "--all", OUT]
        disasm.main()
        sys.exit(0)
    lines = open(OUT).read().splitlines()
    if sys.argv[1:2] == ["func"]:
        # the function containing ADDR: back to the last `push ebp` / `mov ebp, esp` pair, then forward to its ret
        for target in sys.argv[2:]:
            at = next(i for i, ln in enumerate(lines) if ln.startswith(target.zfill(8)))
            s = at
            while not (lines[s].split(None, 1)[1:] == ["push ebp"] and lines[s + 1].endswith("mov ebp, esp")):
                s -= 1
            e = at
            while not (lines[e].split(None, 1)[1:] == ["push ebp"] and lines[e + 1].endswith("mov ebp, esp") and e > at):
                e += 1
            e -= 1
            print(f"== function {lines[s][:8]} contains {target}")
            print("\n".join(lines[s:e + 1]))
        sys.exit(0)
    for target in sys.argv[1:]:
        want = "call    0x" + target.lstrip("0x")
        print(f"== callers of {target}")
        for i, ln in enumerate(lines):
            if ln.endswith(want) or ln.split("  ", 1)[-1].strip() == want.replace("    ", " "):
                print("\n".join(lines[max(0, i - 6):i + 1]))
                print("--")
