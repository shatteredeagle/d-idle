# SPDX-License-Identifier: GPL-3.0-or-later
"""VERIFY-SKRESIST: disassemble a range of the patched game.exe with upstream's disasm.
    python vr_dis.py HEXADDR COUNT"""
import sys
import disasm

va, n = sys.argv[1], sys.argv[2]
sys.argv = ["disasm", "--exe", "/tmp/casttrace/game.exe", va, n]
disasm.main()
