# SPDX-License-Identifier: GPL-3.0-or-later
"""verifysmite: every call of FUN_0057ec10 with the 4 instructions after it; data words equal to given addresses;
skills.txt rows (srvdofunc, srvdofunc table entry) for the named skills.   vs_scan.py SKILL..."""
import os, sys, struct, capstone, pefile
from emu import data_path
import emu
pe = pefile.PE(os.path.join(data_path(), "bin", "game.exe")); base = pe.OPTIONAL_HEADER.ImageBase
img = pe.get_memory_mapped_image(); md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
def fn_start(a):
    while a > base:
        if img[a - base - 1] == 0xcc and img[a - base] != 0xcc: return a
        a -= 1
text = [s for s in pe.sections if s.Characteristics & 0x20000000]
for s in text:
    lo = s.VirtualAddress; data = img[lo:lo + s.Misc_VirtualSize]
    for off in range(len(data) - 5):
        if data[off] == 0xe8 and base + lo + off + 5 + int.from_bytes(data[off+1:off+5], "little", signed=True) == 0x57ec10:
            a = base + lo + off
            after = [f"{i.mnemonic} {i.op_str}" for _, i in zip(range(5), md.disasm(img[a - base:a - base + 40], a))][1:]
            print(f"X {a:08x} fn {fn_start(a):08x} :: " + " ; ".join(after))
# the do-func pointer table: find 0x5ce9f0 as a data word
needle = struct.pack("<I", 0x5ce9f0)
for s in pe.sections:
    lo = s.VirtualAddress; data = img[lo:lo + s.Misc_VirtualSize]; i = data.find(needle)
    while i >= 0:
        print(f"P 005ce9f0 at {base+lo+i:08x} ({s.Name.strip(chr(0).encode()).decode()})"); i = data.find(needle, i + 1)
e = emu.Emu()
t = e.mpq_file("data\\global\\excel\\skills.txt").decode("latin-1").splitlines(); head = t[0].split("\t")
for ln in t[1:]:
    r = dict(zip(head, ln.split("\t")))
    if r.get("skill") in sys.argv[1:]:
        print("S", r["skill"], "Id", r.get("Id"), "srvdofunc", r.get("srvdofunc"), "srvstfunc", r.get("srvstfunc"), "srvmissile", r.get("srvmissile"), "HitShift", r.get("HitShift"), "EType", r.get("EType"), "HitFlags?", r.get("hitflags", "-"), "ResultFlags", r.get("ResultFlags", "-"), "HitClass", r.get("HitClass", "-"))
