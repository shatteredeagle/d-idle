#!/usr/bin/env python3
"""Reading aid (card skshield): disassemble game.exe functions with upstream's disasm.py; output goes to the scratch dir.

    run-on-worker.sh skshield_dis.py 63bef0 535d10 4e9200:120
An argument ADDR runs to the function's end; ADDR:N prints N instructions from ADDR.
"""
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    for a in sys.argv[1:] or ["63bef0"]:
        print(f"== {a}")
        sys.argv = ["disasm.py", *a.split(":")]
        try:
            disasm.main()
        except SystemExit as stop:
            print(f"(stopped: {stop})")
