#!/usr/bin/env python3
"""Reading aid (card verifyskassassin): linear disassembly of game.exe from ADDR for N instructions.

    run-on-worker.sh vsa_dis.py 56c160 140
"""
import os
import sys

import capstone
import pefile

from emu import data_path


def main():
    addr, n = int(sys.argv[1], 16), int(sys.argv[2])
    pe = pefile.PE(os.path.join(data_path(), "bin", "game.exe"), fast_load=True)
    base = pe.OPTIONAL_HEADER.ImageBase
    md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
    rva = addr - base
    data = pe.get_data(rva, n * 15)
    for i, ins in enumerate(md.disasm(data, addr)):
        if i >= n:
            break
        print(f"{ins.address:08x} {ins.mnemonic} {ins.op_str}")


if __name__ == "__main__":
    main()
