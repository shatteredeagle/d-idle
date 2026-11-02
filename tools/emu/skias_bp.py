#!/usr/bin/env python3
"""skias_bp.py: d2d attack_speed / attack_ticks breakpoints, pure arithmetic (no game code).
Mirrors components/rules/combat.hpp effective_speed, attack_speed, attack_ticks.
Usage: skias_bp.py FRAMES RATE WSM [SIAS]   (e.g. 16 256 0)
Prints, per IAS 0..200, the tick count and the IAS where it first changes."""
import sys


def eff(s, cap=120):
    return cap * s // (cap + s) if s > 0 else s


def speed(rate, ias, wsm, sias=0):
    e = max(-85, min(75, eff(ias) - wsm + sias))
    return max(rate * (100 + e) // 100, 1)


def ticks(frames, rate, ias, wsm, sias=0):
    s = speed(rate, ias, wsm, sias)
    return (256 * frames + s - 1) // s


def main():
    frames, rate, wsm = (int(x) for x in sys.argv[1:4])
    sias = int(sys.argv[4]) if len(sys.argv) > 4 else 0
    prev = None
    for ias in range(0, 201):
        t = ticks(frames, rate, ias, wsm, sias)
        if t != prev:
            print(f"ias>={ias} eff={eff(ias)} ticks={t}")
            prev = t


main()
