#!/bin/bash
# monsterai2 (worker host): compile tools/emu/mai2_and_test.cpp by hand against the scratch d2d tree (upstream f8aab0e8, header-only
# use of components/rules/monsters.hpp) and run it on the Andariel cases mai2_six.py wrote.
#   tools/emu/run-on-worker.sh --ttl 1500 mai2_build.sh [cases file]
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/monsterai-src}
S=${D2_SCRATCH:-/tmp/d2agent-monsterai2}
CASES=${1:-$S/andariel_cases.txt}
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
IN="$IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src"
nice c++ -std=c++26 -O1 -include "$SRC/cmake/no_near_far.hpp" -Wno-missing-field-initializers $IN -o "$S/mai2_and_test" "$here/mai2_and_test.cpp"
"$S/mai2_and_test" "$CASES"
