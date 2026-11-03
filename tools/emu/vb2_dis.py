# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyb2: print game.exe's disassembly of the functions named on the command line
(upstream's disasm.py, once per address). Output to stdout; no game file is stored."""
import sys

import disasm

for a in sys.argv[1:]:
    print(f"=== {a}")
    sys.argv = ["disasm.py", a]
    try:
        disasm.main()
    except SystemExit as e:
        print("exit", e)
