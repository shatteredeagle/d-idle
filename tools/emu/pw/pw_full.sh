#!/bin/bash
# prwarp (worker host): the FULL use_warp path against game.exe's FUN_0064dea0 (the verifier's method, verification-warp-arrival-2026-11-12.md)
#   pw_full.sh probe TAG            compile pw_usewarp.cpp against build-pw's libgame into ~/d2d-lab/pw/bin/usewarp-TAG
#   pw_full.sh full BIN SEED [FROM TO]   run it: the hero's final subtile vs game.exe on the pre- and post-arrival windows of the destination
set -u
here=$(cd "$(dirname "$0")" && pwd)
O=$HOME/d2d-lab/pw; mkdir -p $O/bin $O/out
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python; UP=$HOME/d2d-lab/src/tools/emu
py() { (cd $UP && PYTHONPATH=$here/..:$UP nice -n 19 $PY $here/../vb2_tele.py "$@"); }
case $1 in
probe)
  SRC=$HOME/d2d-lab/wt-pw; B=$HOME/d2d-lab/build-pw; IN=""
  for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
  LIBS="$B/components/game/libgame.a $B/_deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so $B/components/install/libinstall_detect.a"
  nice -n 19 c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++23 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o $O/bin/usewarp-$2 $here/pw_usewarp.cpp $LIBS && echo built usewarp-$2 ;;
full)
  bin=$2; seed=$3; D=$O/out/full-$(basename $bin)-$seed-${4:-1}-${5:-39}; rm -rf $D; mkdir -p $D
  nice -n 19 $bin $seed ${4:-1} ${5:-39} $D > $D/probe.log 2>&1; tail -1 $D/probe.log
  py game $D/pre.tsv > $D/game-pre.tsv; py game $D/post.tsv > $D/game-post.tsv
  echo "== $(basename $bin) seed $seed: final vs game.exe on the post-arrival window"; py cmp $D/post.tsv $D/game-post.tsv $D/final.tsv
  echo "== $(basename $bin) seed $seed: final vs game.exe on the pre-arrival window"; py cmp $D/pre.tsv $D/game-pre.tsv $D/final.tsv ;;
esac
