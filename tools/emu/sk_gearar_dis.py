#!/usr/bin/env python3
"""Disassemble the gear attack-rating functions (card skgearar). Reading aid only: output goes to the scratch dir."""
import subprocess
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    for a in sys.argv[1:] or ["57d8a0"]:
        print(f"== {a}")
        sys.argv = ["disasm.py", a]
        disasm.main()
