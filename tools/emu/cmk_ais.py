"""covmonskills2: monsterai2's mai2_ais.py (imported, not edited) with one label changed: mai2's check() zeroes the `keep` act's y
(the move mode), which for FUN_005de6d0 callers is always 2 (d2d's Think has no field, mode 0). A caller that reaches FUN_005de4e0
directly with a mode of its own (the Vulture: 2, 8, 9) has that mode on both sides, so for the AIs in RAWKEEP the mode is kept.

    PYTHONPATH=<upstream tools/emu>:tools/emu D2_MPQ_DIR=<dir> python cmk_ais.py <ai,ai> <cases> <seed> --dump <dir>
Same output files as mai2_ais.py (mai2_cases.inc, mai2_names.inc). Never runs a game."""
import random
import sys

import mai2_ais as X
import mon_ais as M

RAWKEEP = ("Vulture", "BaalCrab", "BaalCrabClone", "Izual", "ReanimatedHorde", "SiegeBeast")   # the AIs whose `keep` mismatches were that token only


def check(e, ai, cases, rng, lines, tally):
    mark = e.brk
    for _ in range(cases):
        e.brk = mark
        s = M.setup(rng, ai)
        command = s["command"]
        out, ctrl, seed = M.run_game(e, s, M.AIS[ai], random.Random(rng.getrandbits(32)))
        s["command"] = command
        out = [x for x in out if x[0] != "scream"]
        home = any(x[0] == "home" for x in out)
        out = [x for x in out if x[0] != "home"]
        s["dying"] = ai == "Fallen" and ("pace", 50, 0) in out
        act, pace, effects, asks = M.outcome(s, out)
        if ai == "BloodRaven" and s.get("home_set"): s["free"][0] = s["home_set"]
        if ai == "FallenShaman" and act[0] == "seq": act[:] = ["skill", act[1], 0, 0, 0, act[5], act[6]]
        if ai == "BloodRaven" and act[0] == "skill" and act[4] == 1:
            fx, fy = s["spots"]["foe"]; act[:] = ["skill", act[1], act[2] - fx, act[3] - fy, 0, act[5], act[6]]
        if ai == "BloodRaven" and act[0] == "circle": act[3] = 0
        if home: act[:] = ["home", 0, 0, 0, 0, act[5], act[6]]
        if act[0] == "keep" and ai not in RAWKEEP: act[3] = 0
        if ai in ("FallenShaman", "BloodRaven", "SkeletonBow") and act[4] == 2: act[4] = 0
        s["stat"][("mon", 0x2b)] = int(s["dying"])
        lines.append(M.dump_case(s, act, pace, effects, asks, ctrl, seed))
        key = (ai, "dying" if s["dying"] else "-", act[0])
        tally[key] = tally.get(key, 0) + 1
        if pace[0] or pace[1]:                                              # also count the pace the think set (percent / steps): which AIs speed up
            pk = (ai, "pace", "%d/%d" % (pace[0], pace[1]))
            tally[pk] = tally.get(pk, 0) + 1


X.check = check

# env-free switch: argv word `distinct` (before --dump) makes the Skill6..8 ids (s["skills68"]) differ from Skill1..5's, so a label
# lookup by id (mon_ais.outcome: list.index) cannot name a slot of the other list. Used to test whether a mismatch class is that label.
_setup = X.setup
DISTINCT = "distinct" in sys.argv


def setup(rng, ai):
    s = _setup(rng, ai)
    if DISTINCT and "skills68" in s:
        used = set(k for k in s["skills"] if k >= 0)
        s["skills68"] = [(k if k < 0 or k not in used else 300 + n) for n, k in enumerate(s["skills68"])]
    if SHIFT:
        s["skills"] = [k + 1000 if k >= 0 else k for k in s["skills"]]
        if "skills68" in s: s["skills68"] = [k + 1000 if k >= 0 else k for k in s["skills68"]]
    return s


SHIFT = "shift" in sys.argv   # every set slot id + 1000: a think that casts a raw id of its own (Catapult Spotter 287, Putrid Defiler 300, ..) cannot collide with a drawn slot id
X.setup = setup

if __name__ == "__main__":
    sys.exit(X.main())
