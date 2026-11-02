# Verifier VERIFYPRICES: linear disassembly of game.exe ranges (card verifyprices).
#   vpr_dis.py START END [START END ...]   (hex addresses)
import os
import sys

import capstone
import pefile

exe = os.path.join(os.environ.get("D2_MPQ_DIR", "/tmp/ug-emu"), "bin", "game.exe")
pe = pefile.PE(exe, fast_load=True)
base = pe.OPTIONAL_HEADER.ImageBase
img = pe.get_memory_mapped_image()
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
args = sys.argv[1:]
for i in range(0, len(args), 2):
    a, b = int(args[i], 16), int(args[i + 1], 16)
    print(f"== {a:08x}..{b:08x}")
    for ins in md.disasm(img[a - base:b - base], a):
        print(f"{ins.address:08x}  {ins.mnemonic} {ins.op_str}")
