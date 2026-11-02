# SPDX-License-Identifier: GPL-3.0-or-later
"""skbarb: Howl's three skill-record readers run natively, and the Skills.txt parameter layout of the record.

    run-on-worker.sh skbarb_howl.py

FUN_004efc20(skill) = record +0x14c; FUN_004cc7c0(skill, lvl) = +0x150 + (lvl - 1) x +0x154;
FUN_004efcb0(skill, lvl) = +0x158 + (lvl - 1) x +0x15c (the three FUN_005aafb0 calls: Howl's hit 17).
Prints each against Skills.txt Param2..Param6 (record +0x14c..+0x15c, Param1 at +0x148) and d2d's expression
(fight.cpp:2932: max(Param3 + (L - 1) x Param4, 25) frames; sum clvl + calc1(L) + L). `S` lines, one `SUMMARY`."""
import skill_damage


def main():
    o = skill_damage.Oracle()
    e = o.e
    rec = o.skills + 130 * skill_damage.RECORD
    par = [e.r32(rec + 0x148 + 4 * k) for k in range(8)]
    print("S Howl record +0x148..+0x164 (Param1..Param8):", par)
    bad_a = bad_b = bad_c = 0
    for lvl in range(1, 41):
        a = e.call(0x4efc20, ecx=130)
        b = e.call(0x4cc7c0, ecx=130, edx=lvl)
        c = e.call(0x4efcb0, ecx=130, edx=lvl)
        pa, pb, pc = par[1], par[2] + (lvl - 1) * par[3], par[4] + (lvl - 1) * par[5]
        bad_a += a != pa
        bad_b += b != pb
        bad_c += c != pc
        d2d_len = max(par[2] + (lvl - 1) * par[3], 25)
        d2d_add = 2 * (lvl - 1)            # calc1 = par1 x (lvl - 1)
        print(f"S L{lvl}: FUN_4efc20 {a}, FUN_4cc7c0 {b}, FUN_4efcb0 {c}; d2d length {d2d_len}; level sum game adds {a} + clvl + L, d2d adds {d2d_add} + clvl + L")
    print(f"SUMMARY: record params {par}; 40 levels: FUN_4efc20 vs Param2 {bad_a} differ, FUN_4cc7c0 vs Param3 + (L-1) Param4 {bad_b}, FUN_4efcb0 vs Param5 + (L-1) Param6 {bad_c}")
    # the other Barbarian rows' parameter words, for the trace's table
    for sk in range(126, 156):
        r = o.skills + sk * skill_damage.RECORD
        print("S params", sk, [e.r32(r + 0x148 + 4 * k) for k in range(8)])


main()
