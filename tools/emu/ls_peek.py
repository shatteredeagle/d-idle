# SPDX-License-Identifier: GPL-3.0-or-later
"""Print dwords of game.exe's mapped image at the given virtual addresses (agent LEVELSPAWN). Read-only.

  python ls_peek.py 0x6ce270 0x6ce280        # via tools/emu/ls_run.sh (needs the scratch MPQ dir with bin/game.exe)
"""
import os
import sys
import struct

import pefile

exe = os.path.join(os.environ["D2_MPQ_DIR"], "bin", "game.exe")
pe = pefile.PE(exe, fast_load=True)
base = pe.OPTIONAL_HEADER.ImageBase
for arg in sys.argv[1:]:
    va = int(arg, 0)
    data = pe.get_data(va - base, 4)
    print(f"{va:#x}: {struct.unpack('<I', data)[0]:#x}")
