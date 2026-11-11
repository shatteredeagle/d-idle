"""decomp9 (worker host): list the instructions of game.exe 1.14d that touch unit flag word +0xc4 with a mask containing the given bits.
    python d9_flagscan.py 0x20000 0x4000000        (run with the upstream tools/emu venv; needs D2_MPQ_DIR with bin/game.exe; capstone)"""
import os, sys, pefile
from capstone import Cs, CS_ARCH_X86, CS_MODE_32
pe = pefile.PE(os.path.join(os.environ["D2_MPQ_DIR"], "bin", "game.exe"), fast_load=True)
base = pe.OPTIONAL_HEADER.ImageBase
sec = [s for s in pe.sections if s.Name.startswith(b".text")][0]
code = sec.get_data(); va = base + sec.VirtualAddress
md = Cs(CS_ARCH_X86, CS_MODE_32); md.skipdata = True
bits = [int(a, 0) for a in sys.argv[1:]]
n=0
for i in md.disasm(code, va):
    n+=1
    if "0xc4]" in i.op_str:
        for b in bits:
            if hex(b) in i.op_str:
                print("%#x %s %s" % (i.address, i.mnemonic, i.op_str)); break

print("instructions", n, "text", hex(va), len(code))
