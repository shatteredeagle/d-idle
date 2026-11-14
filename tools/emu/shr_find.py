"""SHRINE (decomp17, worker host): locate a string in game.exe and list the instructions that push/load its address, with the owner function start.
    python shr_find.py shrines objgroup     (each arg: a string searched case-insensitively in the loaded image)
Run: tools/emu/run-on-worker.sh shr_find.py shrines"""
import os, sys, re, pefile
from capstone import Cs, CS_ARCH_X86, CS_MODE_32
pe = pefile.PE(os.path.join(os.environ["D2_MPQ_DIR"], "bin", "game.exe"), fast_load=True)
base = pe.OPTIONAL_HEADER.ImageBase
img = pe.get_memory_mapped_image()
sec = [s for s in pe.sections if s.Name.startswith(b".text")][0]
code = sec.get_data(); va = base + sec.VirtualAddress
md = Cs(CS_ARCH_X86, CS_MODE_32); md.skipdata = True
ins = list(md.disasm(code, va))
for s in sys.argv[1:]:
    hits = [m.start() for m in re.finditer(re.escape(s.encode()), img, re.I)]
    print("string", s, [hex(base + h) for h in hits][:8])
    for h in hits[:8]:
        a = base + h
        for i in ins:
            if hex(a) in i.op_str:
                print("  xref %#x %s %s" % (i.address, i.mnemonic, i.op_str))
