# SPDX-License-Identifier: GPL-3.0-or-later
"""covpal: entries of game.exe's srvdofunc table (0x7322b0, the base upstream's skills.md names) for the covpal rows.
    tools/emu/run-on-worker.sh covpal_tab.py > /tmp/d2agent-covpal/tab.txt"""
import skill_damage

o = skill_damage.Oracle()
for i in (2, 18, 65, 66, 67, 73, 79, 80, 81, 82):
    print(i, hex(o.e.r32(0x7322b0 + 4 * i)))
