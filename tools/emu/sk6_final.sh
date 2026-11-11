#!/bin/bash
# decomp6 (worker host): final head of fid/missile-last-unit: rebuild, the Lightning differential on two seeds, the neighbours
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
git -C $T log --oneline -1
cd $B && nice -n 19 ninja -j3 test_light_pierce test_next_hit test_game_missiles test_monsters test_game test_combat 2>&1 | tail -1
for s in "1 400" "7 600"; do bash $HERE/d6_base.sh light $s 2>&1 | sed -n 2,10p; done
echo "== next_hit"; $B/tests/test_next_hit --next-hit 1 400 > $O/nh2.d 2>/dev/null; (cd $T/tools/emu && uv run python next_hit.py $O/nh2.d 2>/dev/null > $O/nh2.g); grep -E "^[HT]" $O/nh2.d | diff $O/nh2.g - | wc -l
for t in test_game_missiles test_monsters test_combat; do $B/tests/$t > $O/$t.out 2>&1; echo "$t rc=$?"; done
(cd $T && $B/tests/test_game > $O/test_game.out 2>&1; echo "test_game rc=$?"; tail -2 $O/test_game.out)
