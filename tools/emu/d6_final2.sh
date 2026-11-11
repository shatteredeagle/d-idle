#!/bin/bash
# decomp6 (worker host): the lines the first runs did not capture: moves.py in full, and upstream's cast_mana differential
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
git -C $T log --oneline -1
cd $T/tools/emu
echo "== moves.py"; uv run python moves.py 2>&1 | grep -v WARNING
cd $B && nice -n 19 ninja -j2 test_cast_mana 2>&1 | tail -1
cd $T/tools/emu
echo "== cast_mana gen 1"; uv run python cast_mana.py gen 1 > $O/cm.tsv 2>/dev/null && uv run python cast_mana.py game $O/cm.tsv > $O/cm.g 2>/dev/null && $B/tests/test_cast_mana --cast-mana $O/cm.tsv | grep -E '^[0-9]+ ' > $O/cm.d; wc -l < $O/cm.g; diff $O/cm.g $O/cm.d | wc -l
