# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint: disassemble FUN_005a5e50 (the only caller of the +0x160 setter) and FUN_00584870.

    tools/emu/run-on-worker.sh wpv_dis.py > $D2_SCRATCH/dis.txt
"""
import sys

import disasm

for a in ("5a5e50", "584870"):
    print("=====", a)
    sys.argv = ["disasm.py", "--exe", __import__("os").path.join(__import__("os").environ.get("D2_EMU_DIR", __import__("os").path.expanduser("~/d2d-lab/emu")), "casttrace", "game.exe"), a]
    try:
        disasm.main()
    except SystemExit:
        pass
