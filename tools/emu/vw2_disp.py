# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: the dispatcher that indexes the 4-entry table holding FUN_005a5e50 (index 4 of 0x6e2218),
its start, its callers and the 8 instructions before each call. Addresses and mnemonics only.

    tools/emu/run-on-worker.sh vw2_disp.py > $D2_SCRATCH/disp.txt
"""
import capstone
import pefile
from capstone import x86

EXE = "/tmp/casttrace/game.exe"
pe = pefile.PE(EXE)
base = pe.OPTIONAL_HEADER.ImageBase
t = [s for s in pe.sections if s.Name.rstrip(b"\0") == b".text"][0]
data, tva = t.get_data(), base + t.VirtualAddress
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True


def dis(a, n):
    out = []
    for ins in md.disasm(data[a - tva:a - tva + 0x200], a):
        out.append((ins.address, ins.mnemonic, ins.op_str))
        if len(out) >= n:
            break
    return out


# find the prologue before 0x5a6031: scan back for 55 8b ec / 83 ec / 6a ff
start = None
for a in range(0x5a6031, 0x5a5fa0, -1):
    if data[a - tva:a - tva + 3] == b"\x55\x8b\xec" and (data[a - tva - 1] in (0xc3, 0xcc) or data[a - tva - 3] == 0xc2):
        start = a
        break
print(f"function start guess {start:08x}" if start else "no prologue found")
if start is None:
    start = 0x5a6020
print("== body (first 70 instructions)")
for a, m, o in dis(start, 70):
    print(f"{a:08x} {m} {o}")
print("== callers (rel32) of the start")
off = 0
rows = []
while off < len(data):
    n = 0
    for ins in md.disasm(data[off:off + 0x4000], tva + off):
        n += 1
        off = ins.address - tva + ins.size
        if ins.mnemonic == "call" and ins.operands and ins.operands[0].type == x86.X86_OP_IMM and ins.operands[0].imm == start:
            rows.append(ins.address)
    if n == 0:
        off += 1
for r in rows:
    print(f"--- call at {r:08x}")
    for a, m, o in dis(r - 30, 14):
        print(f"   {a:08x} {m} {o}")
print(f"callers: {len(rows)}")
