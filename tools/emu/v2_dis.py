#!/usr/bin/env python3
"""verifydmg (independent check of the monster damage roll): disassemble game.exe functions with upstream's disasm.py.

    run-on-worker.sh vd_dis.py 57b420 5a89a0
"""
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    for a in sys.argv[1:]:
        print(f"== {a}")
        sys.argv = ["disasm.py", a]
        disasm.main()
