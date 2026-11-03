#!/usr/bin/env python3
"""verifydmg3 (third independent check of the monster damage roll): disassemble game.exe ranges with upstream's disasm.py.
Each argument is ADDR or ADDR:COUNT (instructions).

    run-on-worker.sh w3_dis.py 57c6c0 5c4d90:40
"""
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    for a in sys.argv[1:]:
        print(f"== {a}")
        sys.argv = ["disasm.py"] + a.split(":")
        try:
            disasm.main()
        except SystemExit as e:
            print(f"exit {e}")
