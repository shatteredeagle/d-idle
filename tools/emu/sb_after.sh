#!/bin/bash
# smiteblock (the worker host): the "after the fix" run of sb_d2d_test. Copies upstream's rules headers of the scratch
# pair into $D2_SCRATCH/patched/, applies the one candidate fix with sed, prints the diff, builds sb_d2d_test_after
# against it (the patched directory first on the include path) and runs it. Nothing in the upstream checkout is touched.
#   F: a Smite blow is never blocked: `if (!swing.smite && target.block > 0 && rng(100) < target.block)` (player_blow)
#   tools/emu/run-on-worker.sh --ttl 1500 sb_after.sh 20000 7
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${B:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:-/tmp/d2agent-smiteblock}
P=$OUT/patched
mkdir -p "$P/rules"
cp "$SRC"/components/rules/* "$P/rules/"
sed -e 's/    if (target.block > 0 \&\& rng(100) < target.block) { blow.blocked = true; return blow; }/    if (!swing.smite \&\& target.block > 0 \&\& rng(100) < target.block) { blow.blocked = true; return blow; }/' \
    "$SRC/components/rules/combat.hpp" > "$P/rules/combat.hpp"
diff -u "$SRC/components/rules/combat.hpp" "$P/rules/combat.hpp" || true
IN="-I$P/rules"
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/sb_d2d_test_after" "$here/sb_d2d_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built "$OUT/sb_d2d_test_after"
"$OUT/sb_d2d_test_after" "${1:-20000}" "${2:-7}"
