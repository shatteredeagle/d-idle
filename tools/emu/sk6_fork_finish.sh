#!/bin/bash
# decomp6 (worker host): turn the scratch edits into the two commits of fid/missile-last-unit (local, then pushed to the private remote)
. "$(cd "$(dirname "$0")" && pwd)/d6_env.sh"
HERE=$(cd "$(dirname "$0")" && pwd)
cd $T
git switch -q fid/missile-last-unit
id=(-c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com)
git stash list >/dev/null  # no stash is ever used
# 1. tooling: rename the oracle and its d2d side
git checkout -q -- components/game/fight 2>/dev/null   # park the claim: the first commit is the oracle alone
rm -f tools/emu/sk6_light_hit.py tools/emu/sk6_light_sum.py
sed -e 's/sk6_light_hit.py/light_pierce.py/g' -e 's/decomp6: next_hit.py with this row/next_hit.py with the lightningbolt row/' $HERE/sk6_light_hit.py > tools/emu/light_pierce.py
sed -e 's/sk6_light_hit.py/light_pierce.py/g' -e 's/ (decomp6)//' -e 's/`--next-hit SEED N`: one Nova or Frost Nova cast (the 64-missile ring,/`--light-hit SEED N`: one Lightning cast (the `lightningbolt` row, LastCollide,/' -e 's/NextHit rows) on 1..6 standing monsters of sizes 1..3, through the/CollideKill 0, Size 2) on 1..6 standing monsters of sizes 1..3, through the/' $HERE/sk6_test_light_pierce.cpp > tests/game/test_light_pierce.cpp
python3 - <<'PY'
import re
p="tools/emu/README.md"
s=open(p).read()
row="| A Lightning cast on a pack in its line (one unit a frame, the last unit skipped: FUN_005ae1f0, FUN_005a8730, FUN_005adf10) | `../../build/tests/test_light_pierce --light-hit 1 400 > d2d.txt; uv run python light_pierce.py d2d.txt > g.txt; grep -E '^[HT]' d2d.txt \\| diff g.txt -` | no output |\n"
key="| A Blessed Hammer hit"
assert key in s
s=s.replace(key,row+key,1)
open(p,"w").write(s)
PY
git add tools/emu/light_pierce.py tests/game/test_light_pierce.cpp tools/emu/README.md
git "${id[@]}" commit -q -F - <<MSG
tests: Lightning on a pack, d2d against game.exe's own hit code

tools/emu/light_pierce.py replays the contacts d2d's Lightning flight made through FUN_005a8850 and FUN_005adf10; test_light_pierce is the d2d side.
Agent: Claude Code
MSG
# 2. the claim
python3 $HERE/sk6_patch_lastcollide.py $T
python3 - <<'PY'
p="docs/research/re/missiles.md"
s=open(p).read()
key="## Pierce — FUN_005ada80"
add="""## One unit a frame, and the last unit skipped — FUN_005ae1f0, FUN_005a8730

A missile whose row has LastCollide and not NextHit (Lightning, `lightningbolt`)
is handed ONE unit a frame: FUN_005ae1f0 stops at the first unit FUN_00641cb0
finds in the subtiles the frame entered that FUN_005a8730 passes (every unit
but the one the missile struck last, FUN_0064a470), and FUN_005adf10 makes
that unit the last one (FUN_0064a400, 0x5adfae). So two monsters in one frame
are struck on two frames at most, and a monster the missile left for another
may be struck again. `Fight::skill_missile_hits` keeps `last_hit` for these
rows as it does for NextHit rows (no `struck` list); tools/emu/light_pierce.py
and test_light_pierce check 2,584 casts. The order of units found in one
subtile is the monster order here; game.exe's is its room's unit list
(not read).

"""
assert key in s
s=s.replace(key,add+key,1)
open(p,"w").write(s)
PY
git add components/game/fight docs/research/re/missiles.md
git "${id[@]}" commit -q -F - <<MSG
Lightning hits one unit a frame and skips the last (FUN_005ae1f0)

A LastCollide row without NextHit struck every touching monster once; game.exe hands one unit a frame to FUN_005adf10 and skips only the last unit hit.
Agent: Claude Code
MSG
git log --oneline -3
