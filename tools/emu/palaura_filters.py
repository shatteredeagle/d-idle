# SPDX-License-Identifier: GPL-3.0-or-later
"""palaura: the aurafilter word (record +0x50), the do functions (srvdofunc) and the passive / aura state ids of every
skill row that has an aurafilter, as game.exe loaded them (patched 1.14d tables).

    tools/emu/run-on-worker.sh palaura_filters.py

Prints `id name filter(hex) aurastate auratargetstate` (65535 = none), then the bit table of the filter words in use, with
the bits decoded in FUN_0056b3e0 (palaura-trace.md section 2)."""
import monstats
import skill_tips

e = skill_tips.boot()
tables = e.r32(0x744304)
skills, count = e.r32(tables + 0xb98), e.r32(tables + 0xba0)
names = [r.get("skill", "") for r in monstats.table("skills")]
used = {}
for s in range(count):
    r = skills + s * 0x23c
    f = e.r32(r + 0x50)
    if f:
        used.setdefault(f, []).append(s)
        print(s, names[s] if s < len(names) else "?", hex(f), "aurastate", e.r16(r + 0x80), "auratargetstate", e.r16(r + 0x82))
print("filter words in use:")
for f, ids in sorted(used.items()):
    bits = [hex(1 << b) for b in range(32) if f >> b & 1]
    print(hex(f), "bits", " ".join(bits), "skills", ids)
