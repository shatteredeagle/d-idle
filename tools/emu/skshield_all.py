#!/usr/bin/env python3
"""Reading aid (card skshield): write every executable instruction of game.exe to a file in the scratch dir, one per line.

    run-on-worker.sh skshield_all.py /tmp/d2agent-skshield/all.txt
Then grep the file (an immediate, a table address, an instruction shape).
"""
import sys

import disasm  # upstream tools/emu/disasm.py

if __name__ == "__main__":
    sys.argv = ["disasm.py", "--all", sys.argv[1]]
    disasm.main()
