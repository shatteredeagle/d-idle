# SPDX-License-Identifier: GPL-3.0-or-later
"""skdefenders: list the call sites (E8 rel32) of game.exe 1.14d functions.

    run-on-worker.sh skdef_callers.py 0x572840 0x57be00 ...
"""
import os
import struct
import sys

sys.path.insert(0, os.getcwd())
import emu


def main():
    e = emu.Emu()
    lo, hi = 0x401000, 0x6a0000
    blob = bytes(e.mu.mem_read(lo, hi - lo))
    targets = [int(a, 16) for a in sys.argv[1:]]
    for t in targets:
        sites = []
        i = blob.find(b"\xe8")
        while i != -1 and i + 5 <= len(blob):
            rel = struct.unpack_from("<i", blob, i + 1)[0]
            if (lo + i + 5 + rel) & 0xFFFFFFFF == t:
                sites.append(lo + i)
            i = blob.find(b"\xe8", i + 1)
        print(hex(t), "called from", " ".join(hex(s) for s in sites))


if __name__ == "__main__":
    main()
