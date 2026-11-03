#!/usr/bin/env python3
"""covstates2: write the whole game.exe disassembly (disasm.py --all) to the scratch dir, for grepping."""
import os
import sys

import disasm

out = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "all.txt")
sys.argv = ["disasm.py", "--all", out]
disasm.main()
print("wrote", out)
