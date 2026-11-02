# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: which C->S message numbers reach the two handlers that call the hostile dispatcher
(0x54c2a0, length 7, and 0x54c300, length 6)? Looks for the handler address as a dword in any section, then
prints the position of that slot in its run of text pointers (= the message number if it is the handler table).

    tools/emu/run-on-worker.sh vw2_op.py > $D2_SCRATCH/op.txt
"""
import struct

import pefile

pe = pefile.PE("/tmp/casttrace/game.exe")
base = pe.OPTIONAL_HEADER.ImageBase
secs = {s.Name.rstrip(b"\0").decode(): (base + s.VirtualAddress, s.get_data()) for s in pe.sections}
tlo = secs[".text"][0]
thi = tlo + len(secs[".text"][1])
for h in (0x54c2a0, 0x54c300, 0x54c5d0):
    pat = struct.pack("<I", h)
    for nm, (va, d) in secs.items():
        i = d.find(pat)
        while i >= 0:
            a = va + i
            lo = a
            while lo - 4 >= va and tlo <= struct.unpack_from("<I", d, lo - 4 - va + (va - va))[0] < thi:
                lo -= 4
            print(f"handler {h:08x} in {nm} at {a:08x}: run starts {lo:08x}, slot index {(a - lo) // 4}")
            i = d.find(pat, i + 1)
