# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: second, independent run of game.exe's 0x49 refusal (FUN_0054d750 -> FUN_0054c5d0 -> FUN_0055b6c0,
hostile setter FUN_0055b720 native) over my own case set, with the CLOCK FREE OVER THE FULL 32-BIT RANGE (the earlier
runs kept it in 70,000..2^31) and the field read back after every case.

    tools/emu/run-on-worker.sh --ttl 3000 vw2_game.py 23 100000 > $D2_SCRATCH/vw2_game.txt 2>&1
    (writes $D2_SCRATCH/vw2_cases.txt and $D2_SCRATCH/vw2_game_out.txt)

Per case: ticks are mod 2^32. Level changes (arrivals) run nothing in game.exe (whole-.text scan, vw2_xref.py), so the
game side only replays the hostile declarations and then the 0x49. Rules compared against the game result:
  A  elapsed since the last hostile declaration (0 = never) < 10000 (mod 2^32), or target not active
  B  field + 10000 (mod 2^32) > now (unsigned)            or not active
  C  same, signed compare                                  or not active
  D  d2d: elapsed since the last arrival < 10000          or not active
Classes: (1) common (now >= 20000 after every event, no event in the future), (2) wrap (an event time wrapped below
the clock), (3) early (now < 10000: field 0 is 'recent').
"""
import random
import sys

import wp_refuse

M = 1 << 32


def gen(n, seed):
    rng = random.Random(seed)
    out = []
    for _ in range(n):
        kind = rng.choice(("common", "common", "common", "wrapclock", "early"))
        if kind == "common":
            now = rng.randrange(40_000, M)
        elif kind == "wrapclock":
            now = rng.randrange(0, 60_000) if rng.random() < 0.5 else rng.randrange(M - 60_000, M)
        else:
            now = rng.randrange(0, 20_000)
        pick = lambda: rng.choice((rng.randrange(0, 12_000), rng.randrange(0, 1_000_000), rng.randrange(0, 20_000_000)))
        levels = sorted((pick() for _ in range(rng.randrange(1, 4))), reverse=True)
        hostiles = sorted((pick() for _ in range(rng.choice((0, 0, 1, 1, 2, 3)))), reverse=True)
        out.append((kind, now, int(rng.random() < 0.88), [(now - d) % M for d in levels], [(now - d) % M for d in hostiles]))
    return out


def main():
    seed, n = int(sys.argv[1]), int(sys.argv[2])
    cases = gen(n, seed)
    import os
    scratch = os.environ.get("D2_SCRATCH", ".")
    with open(scratch + "/vw2_cases.txt", "w") as f:
        for i, (_k, now, active, lv, hs) in enumerate(cases):
            f.write(" ".join(str(x) for x in (i, now, active, len(lv), *lv, len(hs), *hs)) + "\n")
    o = wp_refuse.Oracle()
    e = o.e
    res = open(scratch + "/vw2_game_out.txt", "w")
    rules = open(scratch + "/vw2_rules_out.txt", "w")   # per case: id, rule D (d2d reading), rule B (game reading)
    stat = {}
    field_bad = 0
    ex = []

    def bump(k, key):
        stat.setdefault(k, {}).setdefault(key, 0)
        stat[k][key] += 1

    for i, (kind, now, active, lv, hs) in enumerate(cases):
        g = o.refused(now, active, hs)
        field = e.r32(o.pcdata + 0x160)
        want = hs[-1] if hs else 0
        if field != want:
            field_bad += 1
            if len(ex) < 5:
                ex.append(("field", i, now, hs, field))
        res.write(f"{i} {g}\n")
        last = hs[-1] if hs else 0
        a = 1 if (not active or (now - last) % M < 10_000) else 0
        b = 1 if (not active or ((last + 10_000) % M) > now) else 0
        s32 = lambda v: v - M if v >= 1 << 31 else v
        c = 1 if (not active or s32((last + 10_000) % M) > s32(now)) else 0
        d = 1 if (not active or (now - lv[-1]) % M < 10_000) else 0
        rules.write(f"{i} {d} {b}\n")
        for k in (kind, "all"):
            bump(k, "n")
            bump(k, "game_refused") if g else None
            for nm, r in (("A", a), ("B", b), ("C", c), ("D_d2d", d)):
                if r != g:
                    bump(k, nm + "_differs")
                    if len(ex) < 12 and nm in ("A", "B") and k == "all":
                        ex.append((nm, i, kind, now, active, hs, lv, g))
    res.close()
    rules.close()
    print("seed", seed, "n", n, "field != last declaration tick (or 0):", field_bad)
    for k in sorted(stat):
        print(k, dict(sorted(stat[k].items())))
    for x in ex:
        print("example", x)


main()
