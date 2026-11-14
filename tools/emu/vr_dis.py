# SPDX-License-Identifier: GPL-3.0-or-later
"""VERIFY-SKRESIST: disassemble a range of the patched game.exe with upstream's disasm.
    python vr_dis.py HEXADDR COUNT"""
import sys
import disasm

va, n = sys.argv[1], sys.argv[2]
sys.argv = ["disasm", "--exe", __import__("os").path.join(__import__("os").environ.get("D2_EMU_DIR", __import__("os").path.expanduser("~/d2d-lab/emu")), "casttrace", "game.exe"), va, n]
disasm.main()
