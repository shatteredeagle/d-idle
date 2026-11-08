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
status)
  echo "== freeze_rules gen 100000 20261101"; uv run python freeze_rules.py gen 100000 20261101 > $O/fr.tsv && uv run python freeze_rules.py game $O/fr.tsv > $O/fr.g && $B/tests/test_freeze_rules --freeze-rules $O/fr.tsv > $O/fr.d && uv run python freeze_rules.py compare $O/fr.tsv $O/fr.g $O/fr.d 2>&1 | grep -v WARNING | tail -6
  echo "== chill_length 100000 1"; uv run python chill_length.py 100000 1 2>/dev/null > $O/cl.g; $B/tests/test_cold_length --cold-length 1 100000 | grep -E '^[BR][0-9]' > $O/cl.d; wc -l < $O/cl.g; diff $O/cl.g $O/cl.d | wc -l
  echo "== chill_length 100000 7"; uv run python chill_length.py 100000 7 2>/dev/null > $O/cl7.g; $B/tests/test_cold_length --cold-length 7 100000 | grep -E '^[BR][0-9]' > $O/cl7.d; wc -l < $O/cl7.g; diff $O/cl7.g $O/cl7.d | wc -l
  echo "== freeze_resume gen 50000 3"; uv run python freeze_resume.py gen 50000 3 > $O/fz.tsv && uv run python freeze_resume.py game $O/fz.tsv > $O/fz.g && $B/tests/test_freeze_resume --freeze-resume $O/fz.tsv > $O/fz.d && uv run python freeze_resume.py compare $O/fz.tsv $O/fz.g $O/fz.d 2>&1 | grep -v WARNING | tail -6
  echo "== static_field 100000 1"; uv run python static_field.py 100000 1 2>/dev/null > $O/sf.g; $B/tests/test_static_field --static-field 1 100000 | grep -E '^[BR][0-9]' > $O/sf.d; wc -l < $O/sf.g; diff $O/sf.g $O/sf.d | wc -l
  ;;
light)
  echo "== lightning pierce (sk6_light_hit.py vs test_light_pierce) seed ${2:-1} n ${3:-400}"
  $B/tests/test_light_pierce --light-hit ${2:-1} ${3:-400} > $O/lp.d 2>$O/lp.err; echo "d2d rc=$?"; grep -c '^T' $O/lp.d
  uv run python sk6_light_hit.py $O/lp.d > $O/lp.g 2>$O/lp.gerr; echo "game rc=$?"; grep -c '^T' $O/lp.g
  grep -E '^[HT]' $O/lp.d | diff $O/lp.g - > $O/lp.diff; echo "differing lines: $(grep -c '^[<>]' $O/lp.diff)"
  python3 $HERE/sk6_light_sum.py $O/lp.g $O/lp.d
  ;;
esac
