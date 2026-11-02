# SPDX-License-Identifier: GPL-3.0-or-later
"""verifysmite: disassemble game.exe ranges and list call sites.  vs_dis.py ADDR [COUNT] | xref TARGET"""
import sys, capstone, pefile
from emu import data_path
import os
pe = pefile.PE(os.path.join(data_path(), "bin", "game.exe"))
base = pe.OPTIONAL_HEADER.ImageBase
img = pe.get_memory_mapped_image()
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
def dis(a, n):
    for i, ins in enumerate(md.disasm(img[a - base:a - base + 16 * n], a)):
        if i >= n: break
        print(f"{ins.address:08x}  {ins.mnemonic} {ins.op_str}")
if sys.argv[1] == "xref":
    tgt = int(sys.argv[2], 16)
    for s in pe.sections:
        if not s.Characteristics & 0x20000000: continue
        lo = s.VirtualAddress; data = img[lo:lo + s.Misc_VirtualSize]
        for off in range(len(data) - 5):
            if data[off] in (0xe8, 0xe9):
                rel = int.from_bytes(data[off + 1:off + 5], "little", signed=True)
                if base + lo + off + 5 + rel == tgt: print(f"{'call' if data[off]==0xe8 else 'jmp'} at {base+lo+off:08x}")
else:
    for spec in sys.argv[1:]:
        a, _, n = spec.partition(":")
        print(f"== {a}"); dis(int(a, 16), int(n or 200))
