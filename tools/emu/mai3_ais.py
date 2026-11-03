"""covmonai2: third pass of the monster think differential (docs/research/covmonai2-trace.md). Imports mai2_ais.py (second pass), mai_ais.py
(first pass) and upstream's mon_ais.py; edits none. What changes against mai2_ais.check, and why:

  * `keep`'s y (the move mode, always 2 for FUN_005de6d0) is normalised to 0 only for the AIs whose d2d port has no such field (Wraith,
    SkeletonBow, as the second pass found). The Vulture's d2d port does carry a mode in y (2, 9, 8); zeroing it was a harness artifact
    (719 of 1,500 Vulture cases and 15 Izual cases in the first run of this card).
  * no Fallen / Blood Raven special cases (those AIs are not run here).

    PYTHONPATH=<upstream tools/emu>:tools/emu D2_MPQ_DIR=<dir> python mai3_ais.py <ai,ai> <cases> <seed> --dump <dir>
Writes <dir>/mai2_cases.inc and mai2_names.inc (the names mai2_patch.py's replay program reads). Never runs a game."""
import os
import random
import sys

import mai2_ais as B
import mai_ais as A
import mon_ais as M

KEEP_Y_ZERO = ("Wraith", "SkeletonBow")
EXTRA = {"Bighead": 0x5eff50, "Vampire": 0x5f4a70}      # section 12 of monster-ai-trace.md: Act 1 and 2 rows upstream mon_ais.py does not list


FLAGS0_IDLE = ("Arach", "Vampire")                 # d2d ports say so in their comments (Arach: monsters.hpp:1643); other AIs keep the harness label
RAW_SKILLS = ("CatapultSpotter", "PutridDefiler")      # cast raw ids (288, 300, 304): a random scene skill id equal to one is a label collision


def setup3(rng, ai):
    s = B.setup(rng, ai)
    if ai == "Ancient":
        # d2d's Ancient measures `apart` from the positions less 2 an axis (self_size / target_size 2 by default), the harness hooks
        # FUN_006416d0 to answer s["dist"] whatever the positions are: put the target dist + 2 east of the monster so both agree.
        mx, my = s["spots"]["mon"]
        s["spots"]["foe"] = (mx + (s["dist"] + 2 if s["dist"] > 0 else 0), my)
    if ai == "WillOWisp":
        # the dance states are 5 (the fives') or 6..12 with word 3 = its place + 1 (1..5 / 1..4): the only values the think writes
        st = s["ctrl"][0]
        if st >= 5:
            s["ctrl"][0] = min(st, 12)
            s["ctrl"][2] = rng.randint(1, 5) if s["ctrl"][0] == 5 else rng.randint(1, 4)
    if ai == "Vampire":                      # scene skill ids may repeat; a cast is labelled by the first slot with its id: make them distinct
        seen = set()
        for n, k in enumerate(s["skills"]):
            if k >= 0 and k in seen: s["skills"][n] = k + 1000 * (n + 1)
            seen.add(s["skills"][n])
    if ai in RAW_SKILLS:
        s["skills"] = [99 if 280 <= k <= 320 else k for k in s["skills"]]
    return s


def check(e, ai, cases, rng, lines, tally):
    mark = e.brk
    for _ in range(cases):
        e.brk = mark
        s = M.setup(rng, ai)
        command = s["command"]
        out, ctrl, seed = M.run_game(e, s, M.AIS[ai], random.Random(rng.getrandbits(32)))
        s["command"] = command
        out = [x for x in out if x[0] not in ("scream", "home")]
        s["dying"] = False
        act, pace, effects, asks = M.outcome(s, out)
        if ai in FLAGS0_IDLE and act[0] == "none" and any(x[0] == "away" and not x[3] and not x[4] for x in out):
            act[:] = ["idle", s["aidel"], 0, 0, 0, act[5], act[6]]     # a flags-0 back-off that cannot set off thinks aidel on (FUN_005defe0; the harness models it for Terror only)
        if act[0] == "keep" and ai in KEEP_Y_ZERO: act[3] = 0
        s["stat"][("mon", 0x2b)] = 0
        lines.append(M.dump_case(s, act, pace, effects, asks, ctrl, seed))
        key = (ai, act[0])
        tally[key] = tally.get(key, 0) + 1


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--") and not os.path.isdir(a)]
    which = args[0]
    cases = int(args[1]) if len(args) > 1 else 1500
    rng = random.Random(int(args[2]) if len(args) > 2 else 1)
    out_dir = sys.argv[sys.argv.index("--dump") + 1]
    for name, addr in {**A.NEW, **EXTRA}.items(): M.AIS.setdefault(name, addr)
    M.ORDER[:] = list(M.AIS)
    M._setup = M.setup
    M.setup = setup3
    e = M.emu.Emu()
    A.patch(e)
    B.patch2(e)
    M.install(e)
    B.extra_hooks(e)
    lines, tally = [], {}
    for ai in which.split(","):
        check(e, ai, cases, rng, lines, tally)
    with open(os.path.join(out_dir, "mai2_cases.inc"), "w") as f:
        f.write("// tools/emu/mai3_ais.py --dump\n" + "".join(f'"{line}",\n' for line in lines))
    M.write_names(os.path.join(out_dir, "mai2_names.inc"))
    for key, n in sorted(tally.items()): print("acts", *key, n)
    print("cases", len(lines))
    return 0


if __name__ == "__main__":
    sys.exit(main())
