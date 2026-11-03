#!/bin/bash
# b2rebase (rebase of b2bundle onto 552acc9a): see docs/research/pr-batch2-body.md; run on the worker host with labrun (hard-coded lab paths below, as the other lab scripts).
# needs tools/emu/{v4_probe.cpp,v4_d2d.cpp,v4_run.py,v3_d2d.cpp,v3_run.py,mdmg2.py,mdmg2_d2d.cpp} (cards verifydmg4, dmgroll) beside it.
# damage roll differentials on the bundle head: exports base and head, builds the harness glue and the probe, runs v4 (real Fight::absorb), v3 and mdmg2.
set -u
here=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-b2bundle-r2; B=$HOME/d2d-lab/build-b2bundle-r2
S=$HOME/d2d-lab/b2bundle-r2-scratch; mkdir -p $S; export D2_SCRATCH=$S
rm -rf $S/t_base $S/t_r4; mkdir -p $S/t_base $S/t_r4
git -C $T archive 552acc9a components tests cmake | tar -x -C $S/t_base
git -C $T archive HEAD components tests cmake | tar -x -C $S/t_r4
echo "head $(git -C $T rev-parse --short HEAD)"
X=$S/t_r4
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$X/$d"; done
(cd $B && nice -n 19 c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O3 -std=c++26 \
	-include "$X/cmake/no_near_far.hpp" -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror \
	-o "$S/v4_probe" "$here/v4_probe.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
echo "probe rc=$?"
for l in base r4; do
	R=""; for d in components/rules components/d2s components/txt components/log; do R="$R -I$S/t_$l/$d"; done
	V4=-DV4_R4; V3=-DV3_R3; M=-DMDMG_R3; [ $l = base ] && { V4=-DV4_BASE; V3=-DV2_BASE; M=-DMDMG_BASE; }
	nice -n 19 c++ $R $V4 -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/v4_d2d_$l" "$here/v4_d2d.cpp"; echo "v4_d2d_$l rc=$?"
	nice -n 19 c++ $R $V3 -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/v3_d2d_$l" "$here/v3_d2d.cpp"; echo "v3_d2d_$l rc=$?"
	nice -n 19 c++ $R $M -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/mdmg2_d2d_$l" "$here/mdmg2_d2d.cpp"; echo "mdmg2_d2d_$l rc=$?"
done
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
echo "== v4_run.py 6000 20261103 (real Fight::absorb)"
timeout 2900 bash $here/run-py.sh v4_run.py 6000 20261103 2>&1 | tee $S/v4_run.txt | tail -60
echo "== mdmg2.py 20000 20261103"
timeout 2900 bash $here/run-py.sh mdmg2.py 20000 20261103 $S/mdmg2_d2d_base $S/mdmg2_d2d_r4 2>&1 | tee $S/mdmg2_run.txt | tail -40
echo "== v3_run.py 10000 314159"
timeout 2900 bash $here/run-py.sh v3_run.py 10000 314159 $S/v3_d2d_base $S/v3_d2d_r4 2>&1 | tee $S/v3_run.txt | tail -40
