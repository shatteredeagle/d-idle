# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint: every .text instruction touching [reg + 0x160] with its function, read/write.

    tools/emu/run-on-worker.sh wpv_scan.py > $D2_SCRATCH/scan.txt

Linear sweep with capstone over the .text of the game.exe the emulator uses.
Prints address, mnemonic, operands, direction. Then, for each WRITE, the
direct callers (E8 rel32) of the containing function start (first
`push ebp` / `sub esp` prologue found by walking back, a heuristic).
"""
import struct
import sys

import capstone
import pefile
from capstone import x86

EXE = sys.argv[1] if len(sys.argv) > 1 else __import__("os").path.join(__import__("os").environ.get("D2_EMU_DIR", __import__("os").path.expanduser("~/d2d-lab/emu")), "casttrace", "game.exe")
OFF = int(sys.argv[2], 16) if len(sys.argv) > 2 else 0x160
pe = pefile.PE(EXE)
base = pe.OPTIONAL_HEADER.ImageBase
text = [s for s in pe.sections if s.Name.rstrip(b"\0") == b".text"][0]
data = text.get_data()
tva = base + text.VirtualAddress
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True
STORES = {"fst", "fstp", "fist", "fistp", "fisttp", "fnstcw", "fstcw", "fnstsw", "fstsw"}
hits = []
off = 0
while off < len(data):
    n = 0
    for ins in md.disasm(data[off:off + 0x4000], tva + off):
        n += 1
        off = ins.address - tva + ins.size
        for k, op in enumerate(ins.operands):
            if op.type == x86.X86_OP_MEM and op.mem.disp == OFF and op.mem.base != 0 and op.mem.base != x86.X86_REG_EIP:
                if ins.mnemonic in ("lea", "nop"):
                    continue
                m = ins.mnemonic
                if m in STORES:
                    d = "W"
                elif m.startswith("cmp") or m.startswith("test"):
                    d = "R"
                elif k == 0 and (m.startswith("mov") or m in ("pop", "set") or m.startswith("set")):
                    d = "W"
                elif k == 0:
                    d = "RW"
                else:
                    d = "R"
                hits.append((ins.address, d, f"{m} {ins.op_str}"))
    if n == 0:
        off += 1
print(f"# {EXE} off=0x{OFF:x} hits={len(hits)}")
for a, d, s in hits:
    print(f"{a:08x} {d:2} {s}")
