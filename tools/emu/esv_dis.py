#!/usr/bin/env python3
"""esverify: disassemble game.exe 1.14d functions by address (reading aid only)."""
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    for a in sys.argv[1:]:
        print(f"== {a}")
        sys.argv = ["disasm.py", a]
        disasm.main()
