#!/usr/bin/env python3
"""verifyskmelee2: disassemble the functions given on the command line with upstream's disasm.py.
   tools/emu/run-on-worker.sh vm2_dis.py 57b420 57b7d0 > $D2_SCRATCH/dis.txt
The output is for reading only; it is never pasted into a doc."""
import sys

import disasm

for addr in sys.argv[1:]:
    print(f"==== {addr}")
    sys.argv = ["disasm.py", addr]
    disasm.main()
