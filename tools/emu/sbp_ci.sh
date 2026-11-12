#!/bin/bash
# prbuildsmite (the worker host): CI parity of the changed files, head vs base: g++ and clang++ with the CI flags plus -Wshadow
# -Wrange-loop-construct -Wconversion; prints the warnings that are new on the head (line numbers dropped), then the CI-flag result.
#   tools/emu/run-on-worker.sh --ttl 1500 sbp_ci.sh
set -e
HEAD=${HEAD:-$HOME/d2d-lab/fork/wt-smiteblock}
BASE=${BASE:-$HOME/d2d-lab/fork/wt-smiteblock-base}
B=${B:-$HOME/d2d-lab/pr/smiteblock-out/head}
OUT=${D2_SCRATCH:-/tmp/d2agent-prbuildsmite}
mkdir -p "$OUT"
cmd() {   # compiler root file extra-flags
    local IN=""
    for d in components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl tests; do IN="$IN -I$2/$d"; done
    nice "$1" -fsyntax-only -DSTORMLIB_NO_AUTO_LINK -DD2D_TEST_SKIPPED=77 $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -std=c++26 -include $2/cmake/no_near_far.hpp -Wall -Wextra -Wpedantic -Wno-missing-field-initializers $4 "$2/$3" 2>&1
}
for cc in g++ clang++; do
    for f in components/game/fight/strike.cpp tests/test_combat.cpp; do
        cmd $cc "$HEAD" $f "-Werror" > "$OUT/ci-$cc-$(basename $f).ci" && echo "$cc $f CI flags: 0 warnings" || { echo "$cc $f CI flags: FAIL"; head -5 "$OUT/ci-$cc-$(basename $f).ci"; }
        for v in base head; do R=$HEAD; [ $v = base ] && R=$BASE
            cmd $cc "$R" $f "-Wshadow -Wrange-loop-construct -Wconversion" | grep "warning:" | sed -E 's/^[^ ]*:[0-9]+:[0-9]+: //' | sort > "$OUT/x-$cc-$(basename $f)-$v" || true
        done
        n=$(comm -13 "$OUT/x-$cc-$(basename $f)-base" "$OUT/x-$cc-$(basename $f)-head" | wc -l)
        echo "$cc $f extra flags: $(wc -l < "$OUT/x-$cc-$(basename $f)-base") warnings on base, $(wc -l < "$OUT/x-$cc-$(basename $f)-head") on head, $n new"
        comm -13 "$OUT/x-$cc-$(basename $f)-base" "$OUT/x-$cc-$(basename $f)-head" | head -5
    done
done
