#!/usr/bin/env python3
"""sb_dbg (card smiteblock): trace FUN_0057ec10's locals while sb_block.py's setup runs one case (debug aid)."""
import emu
from unicorn import UC_HOOK_CODE
from unicorn.x86_const import UC_X86_REG_EBX, UC_X86_REG_EBP, UC_X86_REG_EIP, UC_X86_REG_EAX

e = emu.Emu()
st = {}
for a, n, v in ((0x554200, 1, 1), (0x5a4ee0, 0, 1), (0x622c40, 3, 1), (0x57d9b0, 2, 0), (0x639df0, 2, 0)):
    e.hook(a, (lambda v: lambda e: v)(v), n)
game = e.alloc(0x200)
att, dfd = e.alloc(0x100), e.alloc(0x100)
e.w32(att, 0); e.w32(dfd, 1)
mu = e.mu
seen = []


def code(uc, addr, size, ud):
    if addr in (0x57ec1f, 0x57ec26, 0x57ecbb, 0x57ecc1, 0x57ecd2, 0x57ed3b, 0x57ec7e):
        seen.append((hex(addr), hex(uc.reg_read(UC_X86_REG_EBX)), hex(uc.reg_read(UC_X86_REG_EBP)),
                     hex(e.r32(uc.reg_read(UC_X86_REG_EBP) - 4)), hex(uc.reg_read(UC_X86_REG_EAX))))


mu.hook_add(UC_HOOK_CODE, code, begin=0x57ec10, end=0x57ed50)
r = e.call(0x57ec10, dfd, 0, 0, ecx=game, edx=att)
print(hex(r))
for s in seen: print(s)
