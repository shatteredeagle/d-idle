#!/usr/bin/env python3
"""verifynmlevel: disassemble functions (hex addresses) with upstream's disasm.py, for reading only."""
import sys

import disasm

for addr in sys.argv[1:]:
    print(f"==== {addr}")
    sys.argv = ["disasm.py", addr]
    disasm.main()
