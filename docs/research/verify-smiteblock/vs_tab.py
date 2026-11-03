# SPDX-License-Identifier: GPL-3.0-or-later
"""verifysmite: srvdofunc table entries (base = where 005ce9f0 sits minus 150 words).  vs_tab.py ID..."""
import os, sys, struct, pefile
from emu import data_path
pe = pefile.PE(os.path.join(data_path(), "bin", "game.exe")); base = pe.OPTIONAL_HEADER.ImageBase
img = pe.get_memory_mapped_image(); tab = 0x732508 - 150 * 4
for i in map(int, sys.argv[1:]):
    print(f"T {i} {struct.unpack_from('<I', img, tab + 4 * i - base)[0]:08x}")
