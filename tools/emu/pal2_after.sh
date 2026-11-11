#!/bin/bash
# palskill (the worker host): the "after the fix" binary. Copies upstream's combat.hpp of the scratch pair into
# $D2_SCRATCH/patched/, applies the two candidate fixes with sed, prints the diff, and builds pal2_test_after against
# it (the patched directory first on the include path). Nothing in the upstream checkout is touched.
#   F: Smite's shield damage carries stat 111 (normal damage), as FUN_0057b420 adds it (edx = 0 path)
#   (a first candidate, "Smite rolls to hit", was WRONG: FUN_005ce9f0 ORs the hit bit in at 0x5cebb0; see the trace's dead ends)
#   tools/emu/run-on-worker.sh --ttl 1500 pal2_after.sh
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${1:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:-/tmp/d2agent-palskill}
P=$OUT/patched
mkdir -p "$P/rules"
cp "$SRC"/components/rules/* "$P/rules/"
sed -e 's/fighter.smite_lo = found->second.mindam;/fighter.smite_lo = found->second.mindam + int(stat(kNormalDamage));/' \
    -e 's/fighter.smite_hi = std::max(found->second.maxdam, fighter.smite_lo);/fighter.smite_hi = std::max(found->second.maxdam + int(stat(kNormalDamage)), fighter.smite_lo);/' \
    "$SRC/components/rules/combat.hpp" > "$P/rules/combat.hpp"
diff -u "$SRC/components/rules/combat.hpp" "$P/rules/combat.hpp" || true
IN="-I$P/rules"
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/pal2_test_after" "$here/pal2_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built "$OUT/pal2_test_after"
