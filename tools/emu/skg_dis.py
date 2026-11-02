# SPDX-License-Identifier: GPL-3.0-or-later
"""skground: disassemble game.exe by address through upstream's disasm.py.

    run-on-worker.sh skg_dis.py ADDR [COUNT] [ADDR [COUNT]] ...   (hex ADDR; COUNT optional; pairs are separated by `,`)
    e.g. skg_dis.py 56b7e0 , 56ba90 , 56bad0
"""
import sys

import disasm

groups, cur = [], []
for a in sys.argv[1:]:
    if a == ",":
        groups.append(cur)
        cur = []
    else:
        cur.append(a)
groups.append(cur)
for g in groups:
    if not g:
        continue
    print("=====", " ".join(g))
    sys.argv = ["disasm.py"] + g
    disasm.main()
