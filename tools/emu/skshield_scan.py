#!/usr/bin/env python3
"""Reading aid (card skshield): find the places that pass a stat id (a `push IMM`) to a call a few instructions later.

    run-on-worker.sh skshield_scan.py 15 16          # hex stat ids; prints site, the call target, and the 3 pushes before it
Output goes to the scratch dir. The stat ids: 0x15 / 0x16 = min / max damage, 0x11 / 0x12 = enhanced damage %.
"""
import os
import sys

import capstone
import pefile

from emu import data_path


def main():
    ids = {int(a, 16) for a in sys.argv[1:]} or {0x15, 0x16}
    pe = pefile.PE(os.path.join(data_path(), "bin", "game.exe"), fast_load=True)
    base = pe.OPTIONAL_HEADER.ImageBase
    md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
    md.skipdata = True
    ins = []
    for s in pe.sections:
        if s.Characteristics & 0x20000000:
            ins += [(a, m, o) for a, _, m, o in md.disasm_lite(s.get_data(), base + s.VirtualAddress)]
    for i, (a, m, o) in enumerate(ins):
        if m != "push":
            continue
        try:
            v = int(o, 16)
        except ValueError:
            continue
        if v not in ids:
            continue
        for j in range(i + 1, min(i + 6, len(ins))):
            if ins[j][1] == "call" and ins[j][2].startswith("0x"):
                print(f"{a:08x} push {v:#x} -> {ins[j][0]:08x} call {ins[j][2]}")
                break


if __name__ == "__main__":
    main()
