#!/bin/bash
# prbuildsmite (the worker host): build tools/emu/sbp_probe.cpp against the base and the head of the smite-block branch and run it.
#   tools/emu/run-on-worker.sh --ttl 1500 sbp_run.sh [CASES_PER_CHANCE] [SEED]     (needs the oracle files of vs_oracle.py in $G)
# Base = upstream main (detached worktree), head = local/smite-block-v1. The libraries come from one build of the head ($B); the
# probe only uses inline headers of components/rules, taken from each worktree's own copy. -O1, nice, one compile at a time.
set -e
here=$(cd "$(dirname "$0")" && pwd)
HEAD=${HEAD:-$HOME/d2d-lab/fork/wt-smiteblock}
BASE=${BASE:-$HOME/d2d-lab/fork/wt-smiteblock-base}
B=${B:-$HOME/d2d-lab/pr/smiteblock-out/head}
G=${G:-$HOME/d2d-lab/pr/smiteblock-out}
OUT=${D2_SCRATCH:-/tmp/d2agent-prbuildsmite}
mkdir -p "$OUT"
for v in base head; do
    SRC=$HEAD; [ $v = base ] && SRC=$BASE
    IN=""
    for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
    (cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/sbp_$v" "$here/sbp_probe.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
    echo "== $v ($(git -C "$SRC" log -1 --format=%h))"
    nice "$OUT/sbp_$v" chances "${1:-20000}" "${2:-7}"
    for f in "$G"/game-*.txt; do
        [ -s "$f" ] || continue
        nice "$OUT/sbp_$v" cases < "$f" > "$OUT/$v-$(basename "$f")"
        awk -v name="$(basename "$f")" -v v=$v '$1=="E"{n++; gl+=$3; gb+=$4; dh=($6==1&&$8>0); dl+=dh; if($3!=dh)diff++; bl+=$7; if($3==1&&dh&&$9==0)oob++; if($10>0)pl++} END{printf "%s %s monster cases %d, game lands %d, d2d lands %d, DIFFERENT %d, d2d marks blocked %d (game blocked %d), damage out of game range %d, plain swing hit a blocking defender %d\n", v, name, n, gl, dl, diff, bl, gb, oob, pl}' "$OUT/$v-$(basename "$f")"
    done
done
