# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: immediates and displacements that point at the 4-entry table 0x6e221c..0x6e2228
(and 16 bytes either side), in .text, with the 3 instructions before each hit (addresses/mnemonics only).

    tools/emu/run-on-worker.sh vw2_tab2.py > $D2_SCRATCH/tab2.txt
"""
import capstone
import pefile
from capstone import x86

EXE = "/tmp/casttrace/game.exe"
LO, HI = 0x6e220c, 0x6e2238
pe = pefile.PE(EXE)
base = pe.OPTIONAL_HEADER.ImageBase
t = [s for s in pe.sections if s.Name.rstrip(b"\0") == b".text"][0]
data, tva = t.get_data(), base + t.VirtualAddress
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True
off = 0
prev = []
while off < len(data):
    n = 0
    for ins in md.disasm(data[off:off + 0x4000], tva + off):
        n += 1
        off = ins.address - tva + ins.size
        hit = False
        for op in ins.operands:
            if op.type == x86.X86_OP_IMM and LO <= op.imm <= HI:
                hit = True
            if op.type == x86.X86_OP_MEM and LO <= (op.mem.disp & 0xffffffff) <= HI:
                hit = True
        if hit:
            for p in prev[-3:]:
                print(f"   {p[0]:08x} {p[1]} {p[2]}")
            print(f"{ins.address:08x} {ins.mnemonic} {ins.op_str}   <== table ref")
        prev.append((ins.address, ins.mnemonic, ins.op_str))
        prev = prev[-4:]
    if n == 0:
        off += 1
print("done")
