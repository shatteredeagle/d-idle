#!/bin/bash
# verifyskassassin (the worker host): game.exe's mana check + payment (vsa_mana.py, natively) vs d2d's rules::mana_cost
# (tools/emu/sk_mana_test.cpp compiled by hand against the sorcrest scratch build), Assassin rows 251..280, levels 1..99.
#   tools/emu/run-on-worker.sh --ttl 1500 vsa_diff.sh 20000 20261102
# Output: the oracle summary, then `vsa diff cases N differ M` and the first differing lines (id game-check game-paid | d2d).
# game.tsv prints paid 0 when the check fails (the start refuses), the same convention as sk_mana.py.
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:?}
SRC=$HOME/d2d-lab/pr/sorcrest-d2d
B=$HOME/d2d-lab/pr/sorcrest-out/build
bash "$here/run-py.sh" vsa_mana.py "${1:-20000}" "${2:-20261102}" dump | tail -3
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
if [ ! -x "$S/vsa_d2d" ]; then
  ( cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$S/vsa_d2d" "$here/sk_mana_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a )
fi
"$S/vsa_d2d" --sk-mana "$S/vsa_cases.tsv" > "$S/vsa_d2d.tsv" || true
head -2 "$S/vsa_d2d.tsv"
n=$(wc -l < "$S/vsa_game.tsv")
d=$(diff "$S/vsa_game.tsv" "$S/vsa_d2d.tsv" | grep -c '^<' || true)
echo "vsa diff cases $n differ $d"
diff "$S/vsa_game.tsv" "$S/vsa_d2d.tsv" | head -20 || true
