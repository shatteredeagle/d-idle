"""base43.py (verifymonsterpace): game.exe FUN_00573cb0 stat block (0x573e8b..0x573f2c, the SetStat run that ends in 0x45) run
natively for a zeroed and a 0x7f-filled MonStats row and difficulties 0..2, SetStat FUN_00627260 hooked and recorded."""
import emu
from unicorn.x86_const import UC_X86_REG_EBX, UC_X86_REG_EDI, UC_X86_REG_ESI, UC_X86_REG_ESP, UC_X86_REG_EDX, UC_X86_REG_EBP
e = emu.Emu()
seen = []
e.hook(0x627260, lambda m: seen.append((m.arg(1), m.arg(2))) or 0, 4)
out = {}
for fill in (0, 0x7f):
    row = e.alloc(0x400); e.mu.mem_write(row, bytes([fill]) * 0x400)
    unit = e.alloc(0x100); stack = e.alloc(0x1000)
    for diff in range(3):
        seen.clear()
        e.mu.reg_write(UC_X86_REG_EBX, row); e.mu.reg_write(UC_X86_REG_EDI, diff); e.mu.reg_write(UC_X86_REG_ESI, unit)
        e.mu.reg_write(UC_X86_REG_EDX, 0); e.mu.reg_write(UC_X86_REG_ESP, stack + 0x800); e.mu.reg_write(UC_X86_REG_EBP, stack + 0xc00)
        e.mu.emu_start(0x573e8b, 0x573f2c)
        v43 = [v for st, v in seen if st == 0x43]
        out[(fill, diff)] = (v43, [hex(st) for st, _ in seen])
for k, v in out.items(): print(k, "stat 0x43 set to", v[0], "stats set:", " ".join(v[1]))
