#!/usr/bin/env python3
"""Reading aid (card skshield): list function starts (`push ebp; mov ebp, esp`) between two addresses, and who calls each.

    run-on-worker.sh skshield_funcs.py 62d000 62d460
"""
import os
import sys

import capstone
import pefile

from emu import data_path


def main():
    lo, hi = int(sys.argv[1], 16), int(sys.argv[2], 16)
    pe = pefile.PE(os.path.join(data_path(), "bin", "game.exe"), fast_load=True)
    base = pe.OPTIONAL_HEADER.ImageBase
    md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
    md.skipdata = True
    ins = []
    for s in pe.sections:
        if s.Characteristics & 0x20000000:
            ins += [(a, m, o) for a, _, m, o in md.disasm_lite(s.get_data(), base + s.VirtualAddress)]
    starts = [ins[i][0] for i in range(len(ins) - 1) if ins[i][1:] == ("push", "ebp") and ins[i + 1][1:] == ("mov", "ebp, esp") and lo <= ins[i][0] <= hi]
    calls = {}
    for a, m, o in ins:
        if m == "call" and o.startswith("0x"):
            try:
                t = int(o, 16)
            except ValueError:
                continue
            if t in starts:
                calls.setdefault(t, []).append(a)
    for s in starts:
        print(f"{s:08x} called from {' '.join(f'{c:08x}' for c in calls.get(s, [])[:8])}")


if __name__ == "__main__":
    main()
