#!/bin/bash
# verifycovmissiles (the worker host): own hit-roll differential, game.exe vs the real d2d (before = upstream main with the Paladin
# skill checkout, after = with the SKMISS fix). Run through the wrapper:
#   tools/emu/run-on-worker.sh --ttl 3000 vcm_hit_run.sh build            # once: hit_before, hit_after into $D2_SCRATCH
#   tools/emu/run-on-worker.sh --ttl 3000 vcm_hit_run.sh run N SEED
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-verifycovmissiles}
mkdir -p "$S"
export D2=$HOME/d2d-lab/pr/palskill
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
PY=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python
if [ "$1" = build ]; then
  build() {  # SRC BUILD_DIR BIN
    local SRC=$1 B=$2 BIN=$3 IN=""
    for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
    (cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$BIN" "$HERE/sk_missile_hit_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
    echo built "$BIN"
  }
  build $HOME/d2d-lab/pr/palskill $HOME/d2d-lab/pr/skmana-out/build "$S/vhit_before"
  build $HOME/d2d-lab/pr/skmiss $HOME/d2d-lab/pr/skmiss-out/build "$S/vhit_after"
  exit 0
fi
N=${2:-3000}; SEED=${3:-1}
OUT=$S/hit$SEED
mkdir -p "$OUT"
export PYTHONPATH=$HERE:$D2/tools/emu
cd "$D2/tools/emu"
$PY "$HERE/vcm_hit.py" gen "$N" "$SEED" > "$OUT/cases.tsv"
nice $PY "$HERE/vcm_hit.py" game "$OUT/cases.tsv" > "$OUT/game.tsv" 2> "$OUT/game.err"
tail -3 "$OUT/game.err"
for v in before after; do
  nice "$S/vhit_$v" --missile-hit "$OUT/cases.tsv" 2> "$OUT/$v.err" | grep -E '^[0-9]+ ' > "$OUT/$v.tsv"
  echo "$v: $(wc -l < "$OUT/$v.tsv") lines, DIFF $(diff "$OUT/game.tsv" "$OUT/$v.tsv" | grep -c '^<' || true)"
done
wc -l "$OUT/cases.tsv" "$OUT/game.tsv"
for sk in 41 46 47 51 55 57 60 121; do
  awk -v s=$sk '$2==s{print $1}' "$OUT/cases.tsv" > "$OUT/ids$sk.txt"
  echo "skill $sk: cases $(wc -l < "$OUT/ids$sk.txt") before-differ $(join <(sort "$OUT/ids$sk.txt") <(diff "$OUT/game.tsv" "$OUT/before.tsv" | grep '^<' | awk '{print $2}' | sort) | wc -l) after-differ $(join <(sort "$OUT/ids$sk.txt") <(diff "$OUT/game.tsv" "$OUT/after.tsv" | grep '^<' | awk '{print $2}' | sort) | wc -l)"
done
