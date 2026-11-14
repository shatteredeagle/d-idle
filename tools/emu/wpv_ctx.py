# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint: context of each +0x160 hit, and the direct callers of two functions.

    tools/emu/run-on-worker.sh wpv_ctx.py > $D2_SCRATCH/ctx.txt

Function start = first byte after the nearest 0xCC padding before the hit
(heuristic). Prints that start and the 14 instructions before each hit, 3
after. Then every `call rel32` to FUN_0055b720 and FUN_0055b6c0, with its
own function start.
"""
import struct
import sys

import capstone
import pefile

EXE = __import__("os").path.join(__import__("os").environ.get("D2_EMU_DIR", __import__("os").path.expanduser("~/d2d-lab/emu")), "casttrace", "game.exe")
HITS = [0x421da9, 0x421e77, 0x4b83d5, 0x4b83fb, 0x55b6d4, 0x55b76b, 0x5c8be4, 0x5ce43c, 0x6435b2, 0x646525]
TARGETS = [0x55b720, 0x55b6c0]
pe = pefile.PE(EXE)
base = pe.OPTIONAL_HEADER.ImageBase
text = [s for s in pe.sections if s.Name.rstrip(b"\0") == b".text"][0]
data = text.get_data()
tva = base + text.VirtualAddress
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)


def fstart(a):
    i = a - tva
    while i > 0 and not (data[i - 1] == 0xCC and data[i] != 0xCC):
        i -= 1
    return tva + i


def dis(lo, hi):
    return [x for x in md.disasm(data[lo - tva:hi - tva], lo)]


for h in HITS:
    s = fstart(h)
    print(f"== hit {h:08x} function starts ~{s:08x}")
    seq = dis(s, h + 0x20)
    idx = [k for k, x in enumerate(seq) if x.address == h]
    if not idx:
        print("   (desync)")
        continue
    k = idx[0]
    for x in seq[max(0, k - 14):k + 4]:
        print(f"   {x.address:08x} {x.mnemonic} {x.op_str}")
for t in TARGETS:
    print(f"== callers of {t:08x}")
    for i in range(len(data) - 4):
        if data[i] == 0xE8:
            rel = struct.unpack_from("<i", data, i + 1)[0]
            if tva + i + 5 + rel == t:
                print(f"   call at {tva + i:08x} in function ~{fstart(tva + i):08x}")
