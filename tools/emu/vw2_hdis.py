# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: disassembly of FUN_005a5e50 (0x5a5e50..0x5a6000) and the two packet-handler call sites
0x54c2a0..0x54c360. Addresses and mnemonics only (for my own reading; none of it is pasted into the trace).

    tools/emu/run-on-worker.sh vw2_hdis.py > $D2_SCRATCH/hdis.txt
"""
import capstone
import pefile

pe = pefile.PE("/tmp/casttrace/game.exe")
base = pe.OPTIONAL_HEADER.ImageBase
t = [s for s in pe.sections if s.Name.rstrip(b"\0") == b".text"][0]
data, tva = t.get_data(), base + t.VirtualAddress
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
for lo, hi in ((0x5a5e50, 0x5a6000), (0x54c270, 0x54c370)):
    print(f"===== {lo:08x}..{hi:08x}")
    for ins in md.disasm(data[lo - tva:hi - tva], lo):
        print(f"{ins.address:08x} {ins.mnemonic} {ins.op_str}")
