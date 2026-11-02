#!/usr/bin/env python3
"""Reading aid (card skshield): list the call sites of game.exe functions with the instructions before each call.

    run-on-worker.sh skshield_xref.py 627910 626e10
Prints, per call site, the 7 instructions before it (the pushed arguments). Output goes to the scratch dir.
"""
import os
import sys

import capstone
import pefile

from emu import data_path


def main():
    targets = {int(a, 16) for a in sys.argv[1:]} or {0x627910}
    pe = pefile.PE(os.path.join(data_path(), "bin", "game.exe"), fast_load=True)
    base = pe.OPTIONAL_HEADER.ImageBase
    md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
    md.skipdata = True
    ins = []
    for s in pe.sections:
        if s.Characteristics & 0x20000000:
            ins += [(a, m, o) for a, _, m, o in md.disasm_lite(s.get_data(), base + s.VirtualAddress)]
    for i, (a, m, o) in enumerate(ins):
        if m == "call" and o.startswith("0x"):
            try:
                t = int(o, 16)
            except ValueError:
                continue
            if t in targets:
                print(f"== call {t:08x} at {a:08x}")
                for b, bm, bo in ins[max(0, i - 7):i + 1]:
                    print(f"   {b:08x}  {bm} {bo}")


if __name__ == "__main__":
    main()
