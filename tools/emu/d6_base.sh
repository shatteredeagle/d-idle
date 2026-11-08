#!/bin/bash
# decomp6 (worker host): re-run upstream's own oracles on the scratch tree of upstream main.
#   d6_base.sh missile | status | move | boss
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
cp $HERE/sk6_*.py $T/tools/emu/ 2>/dev/null
cd $T/tools/emu
git -C $T log --oneline -1
set -o pipefail
case $1 in
missile)
  echo "== missile_hit gen 60000 1"; uv run python missile_hit.py gen 60000 1 > $O/mh.tsv && uv run python missile_hit.py game $O/mh.tsv > $O/mh.g && $B/tests/test_missile_hit --missile-hit $O/mh.tsv | grep -E '^[0-9]+ ' > $O/mh.d; wc -l $O/mh.g $O/mh.d | head -2; diff $O/mh.g $O/mh.d | wc -l
  echo "== missile_hit gen 20000 7"; uv run python missile_hit.py gen 20000 7 > $O/mh7.tsv && uv run python missile_hit.py game $O/mh7.tsv > $O/mh7.g && $B/tests/test_missile_hit --missile-hit $O/mh7.tsv | grep -E '^[0-9]+ ' > $O/mh7.d; wc -l $O/mh7.g | head -1; diff $O/mh7.g $O/mh7.d | wc -l
  echo "== missiles.py 1500 7"; uv run python missiles.py 1500 7 | tail -3
  echo "== next_hit 1 400"; $B/tests/test_next_hit --next-hit 1 400 > $O/nh.d 2>/dev/null; uv run python next_hit.py $O/nh.d > $O/nh.g; grep -E '^[HT]' $O/nh.d | diff $O/nh.g - | wc -l; grep -c '^T' $O/nh.d
  ;;
light)
  echo "== lightning pierce (sk6_light_hit.py vs test_light_pierce) seed ${2:-1} n ${3:-400}"
  $B/tests/test_light_pierce --light-hit ${2:-1} ${3:-400} > $O/lp.d 2>$O/lp.err; echo "d2d rc=$?"; grep -c '^T' $O/lp.d
  uv run python sk6_light_hit.py $O/lp.d > $O/lp.g 2>$O/lp.gerr; echo "game rc=$?"; grep -c '^T' $O/lp.g
  grep -E '^[HT]' $O/lp.d | diff $O/lp.g - > $O/lp.diff; echo "differing lines: $(grep -c '^[<>]' $O/lp.diff)"
  python3 $HERE/sk6_light_sum.py $O/lp.g $O/lp.d
  ;;
esac
