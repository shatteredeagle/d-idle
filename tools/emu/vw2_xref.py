# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: independent cross-reference scan of game.exe for the waypoint refusal field.

    tools/emu/run-on-worker.sh vw2_xref.py > $D2_SCRATCH/xref.txt

Different from wpv_scan.py: (a) every dword in every section equal to a target address
(rel32 call/jmp targets, absolute immediates, pointer tables); (b) every instruction whose
operand carries displacement 0x160 INCLUDING lea/add/sub; (c) every `rep movs/stos` and
indirect call/jmp in the setter's and its caller's functions is listed. No code is printed:
addresses and mnemonics only.
"""
import struct
import sys

import capstone
import pefile
from capstone import x86

EXE = "/tmp/casttrace/game.exe"
TARGETS = {0x55b720: "setter FUN_0055b720", 0x5a5e50: "hostile FUN_005a5e50", 0x55b6c0: "reader FUN_0055b6c0",
           0x584870: "TP operate FUN_00584870", 0x6221a0: "pcdata getter FUN_006221a0"}
pe = pefile.PE(EXE)
base = pe.OPTIONAL_HEADER.ImageBase
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True

print("== (a) raw dword scan of every section for each target address")
for s in pe.sections:
    d = s.get_data()
    nm = s.Name.rstrip(b"\0").decode()
    va = base + s.VirtualAddress
    for t, label in TARGETS.items():
        pat = struct.pack("<I", t)
        i = d.find(pat)
        while i >= 0:
            print(f"{nm} {va + i:08x} absolute dword = {t:08x} ({label})")
            i = d.find(pat, i + 1)

text = [s for s in pe.sections if s.Name.rstrip(b"\0") == b".text"][0]
data = text.get_data()
tva = base + text.VirtualAddress
print("== (b) text: rel32 call/jmp to a target, and any operand with disp 0x160")
off = 0
rows160 = []
while off < len(data):
    n = 0
    for ins in md.disasm(data[off:off + 0x4000], tva + off):
        n += 1
        off = ins.address - tva + ins.size
        if ins.mnemonic in ("call", "jmp") and ins.operands and ins.operands[0].type == x86.X86_OP_IMM:
            t = ins.operands[0].imm
            if t in TARGETS:
                print(f"{ins.address:08x} {ins.mnemonic} -> {t:08x} ({TARGETS[t]})")
        for op in ins.operands:
            if op.type == x86.X86_OP_MEM and op.mem.disp == 0x160 and op.mem.base not in (0, x86.X86_REG_EIP):
                rows160.append((ins.address, ins.mnemonic, ins.op_str))
            if op.type == x86.X86_OP_IMM and op.imm == 0x160 and ins.mnemonic in ("add", "sub", "mov", "lea", "push"):
                rows160.append((ins.address, ins.mnemonic + " imm", ins.op_str))
    if n == 0:
        off += 1
print(f"== (c) displacement-or-imm 0x160 hits: {len(rows160)}")
for a, m, o in rows160:
    print(f"{a:08x} {m} {o}")
