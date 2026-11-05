"""covmonai: coverage pass over the Act 1 and 2 monster thinks no earlier card ran (docs/research/covmonai-trace.md).

Reuses monsterai2's harness (mai2_ais.py: scene, hooks, case line, normalisations) and only adds think addresses for AIs that upstream's
mon_ais.py and monsterai's mai_ais.py do not name: SkeletonMage, Vampire, Bighead (MonAI row + 8, mai_table.py). Every other AI in the
list is already in the combined table. The replay is mai2_patch.py's (d2d's rules::mon_think on the recorded game.exe runs).

One change to the case writer: monsterai2 zeroes a `keep` act's y (it is the move mode, always 2 for the wrapper FUN_005de6d0). An AI
that calls FUN_005de4e0 itself (the Vulture) has real modes, so for the names in RAW_KEEP the y is kept as game.exe wrote it.

    PYTHONPATH=<upstream tools/emu>:tools/emu D2_MPQ_DIR=<dir> python cvm_ais.py <ai,ai> <cases> <seed> --dump <dir>
    env CVM_RAWKEEP=Vulture,..  (AIs whose keep y is not normalised; default Vulture)
    env CVM_FOE2=Arach,..       (AIs whose second-target unit label is written as the primary; default none)
Writes <dir>/mai2_cases.inc and mai2_names.inc (the replay's file names). Never runs a game."""
import os
import random
import sys

import mai_ais as A
import mai2_ais as B
import mon_ais as M

A.NEW.update({"SkeletonMage": 0x5f96c0, "Vampire": 0x5f4a70, "Bighead": 0x5eff50})
RAW_KEEP = os.environ.get("CVM_RAWKEEP", "Vulture").split(",")
FOE2 = [x for x in os.environ.get("CVM_FOE2", "").split(",") if x]   # AIs whose unit label foe2 is written as foe (run twice: with and without)


def check(e, ai, cases, rng, lines, tally):
    mark = e.brk
    for _ in range(cases):
        e.brk = mark
        s = M.setup(rng, ai)
        command = s["command"]
        out, ctrl, seed = M.run_game(e, s, M.AIS[ai], random.Random(rng.getrandbits(32)))
        s["command"] = command
        out = [x for x in out if x[0] != "scream"]
        s["dying"] = False
        act, pace, effects, asks = M.outcome(s, out)
        if act[0] == "keep" and ai not in RAW_KEEP: act[3] = 0            # move mode label (mai2 Q5)
        if ai in FOE2 and act[4] == 2: act[4] = 0                         # second target = primary, as ai.cpp feeds it (label only)
        s["stat"][("mon", 0x2b)] = 0
        lines.append(M.dump_case(s, act, pace, effects, asks, ctrl, seed))
        key = (ai, "-", act[0])
        tally[key] = tally.get(key, 0) + 1


B.check = check
_setup = B.setup


def setup(rng, ai):
    """monsterai2 Q7a: unset skill ids all -1 make the harness name the first unset slot for every cast of one. Distinct negative ids
    (still `unset` to game.exe: its tests are `id < 0`) keep the slot labels apart. CVM_SAMEUNSET=1 keeps the old scene."""
    s = _setup(rng, ai)
    if not os.environ.get("CVM_SAMEUNSET"):
        s["skills"] = [k if k >= 0 else -1 - n for n, k in enumerate(s["skills"])]
    return s


B.setup = setup

if __name__ == "__main__":
    sys.exit(B.main())
