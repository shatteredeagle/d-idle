#!/usr/bin/env python3
"""Reading aid (card geararmore): disassemble game.exe functions with upstream's disasm.py; output goes to the scratch dir.

    run-on-worker.sh gar_dis.py 57d9b0 57b7d0
"""
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    for a in sys.argv[1:] or ["57d9b0"]:
        print(f"== {a}")
        sys.argv = ["disasm.py", a]
        disasm.main()
