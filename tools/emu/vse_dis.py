#!/usr/bin/env python3
"""verifystateev2: disassemble game.exe functions given as hex args (reading aid; output to stdout)."""
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    for a in sys.argv[1:]:
        print(f"== {a}")
        sys.argv = ["disasm.py", a]
        disasm.main()
