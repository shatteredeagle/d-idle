# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskmana: print the instructions of the two mana functions (reading aid, output stays in scratch)."""
import emu as emu_module

e = emu_module.Emu()
try:
    import capstone
    md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
    for name, a in (("check", 0x56C160), ("pay", 0x56BFE0)):
        print("==", name, hex(a))
        for i in md.disasm(e.read(a, 0x120), a):
            print("%x %s %s" % (i.address, i.mnemonic, i.op_str))
except ImportError:
    for name, a in (("check", 0x56C160), ("pay", 0x56BFE0)):
        print(name, e.read(a, 0x100).hex())
