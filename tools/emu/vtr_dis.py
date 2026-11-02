# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyteleroom: print the disassembly of the functions the Teleport room rule rests on
(upstream's disasm.py, called once per address). Output goes to stdout; no game file is stored."""
import sys

import disasm

for a in sys.argv[1:]:
    print(f"=== {a}")
    sys.argv = ["disasm.py", a]
    try:
        disasm.main()
    except SystemExit as e:
        print("exit", e)
