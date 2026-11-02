# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: what is the pointer table that holds FUN_005a5e50, and who indexes it?

    tools/emu/run-on-worker.sh vw2_tab.py > $D2_SCRATCH/tab.txt

Prints the contiguous run of .text pointers around the table slot, then every instruction
in .text whose memory operand displacement lies inside that run (the indexers), then a
short disassembly of the indexers' function entry area (addresses and mnemonics only).
"""
import struct

import capstone
import pefile
from capstone import x86

EXE = "/tmp/casttrace/game.exe"
SLOT = 0x6e2228
pe = pefile.PE(EXE)
base = pe.OPTIONAL_HEADER.ImageBase
secs = {s.Name.rstrip(b"\0").decode(): (base + s.VirtualAddress, s.get_data()) for s in pe.sections}
rva, rd = secs[".rdata"]
text_lo = secs[".text"][0]
text_hi = text_lo + len(secs[".text"][1])


def dw(a):
    return struct.unpack_from("<I", rd, a - rva)[0]


def intext(v):
    return text_lo <= v < text_hi


lo = SLOT
while intext(dw(lo - 4)):
    lo -= 4
hi = SLOT
while intext(dw(hi + 4)):
    hi += 4
print(f"table run {lo:08x}..{hi:08x} entries={(hi - lo) // 4 + 1} slot index={(SLOT - lo) // 4}")
for a in range(lo, hi + 4, 4):
    print(f"  [{(a - lo) // 4:3d}] {a:08x} -> {dw(a):08x}{'   <== FUN_005a5e50' if a == SLOT else ''}")

md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True
data = secs[".text"][1]
print("== text operands with displacement inside the run (indexers)")
off = 0
hits = []
while off < len(data):
    n = 0
    for ins in md.disasm(data[off:off + 0x4000], text_lo + off):
        n += 1
        off = ins.address - text_lo + ins.size
        for op in ins.operands:
            if op.type == x86.X86_OP_MEM and lo <= (op.mem.disp & 0xffffffff) <= hi + 4:
                hits.append((ins.address, ins.mnemonic, ins.op_str))
        if ins.address == 0x5a0f47:
            print(f"(ctx) {ins.address:08x} {ins.mnemonic} {ins.op_str}")
    if n == 0:
        off += 1
for a, m, o in hits:
    print(f"{a:08x} {m} {o}")
print("== disassembly 0x5a0f30..0x5a0f60 (the add ecx,0x160)")
code = data[0x5a0f30 - text_lo:0x5a0f60 - text_lo]
for ins in md.disasm(code, 0x5a0f30):
    print(f"{ins.address:08x} {ins.mnemonic} {ins.op_str}")
