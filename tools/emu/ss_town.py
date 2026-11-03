"""seedspawn: what game.exe's room population makes in Rogue Encampment (level 1), per room, seeds given as args.

    tools/emu/run-on-worker.sh ss_town.py 1 2 3

Prints monsters.py's dump (class@x,y[m<mode>]/leader) for level 1; the `monsters` kind of diff_drlg.py compares it with
drlg-dump's `monsters`, which prints nothing for a town (NPCs are left out of both, anything else is not)."""
import sys

import drlg
import monsters

e = drlg.boot()
for arg in sys.argv[1:]:
    seed = int(arg, 0)
    print(f"seed {seed} level 1:")
    print(monsters.dump(e, seed, 1) or "(no monsters)")
