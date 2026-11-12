#!/bin/bash
# dmgroll r4: build the author's two earlier harnesses (v3_d2d.cpp, mdmg2_d2d.cpp: monster_blow + absorb_blows, header-only) against the exported
# base, r2 and r4 trees ($D2_SCRATCH/t_*, from w4_git.sh), then run v3_run.py and mdmg2.py on them.
#   tools/emu/run-on-worker.sh --ttl 3000 w4_old.sh
set -eu
S=${D2_SCRATCH:?}
here=$(cd "$(dirname "$0")" && pwd)
for l in base r2 r4; do
	IN=""
	for d in components/rules components/d2s components/txt components/log; do IN="$IN -I$S/t_$l/$d"; done
	V=""; M=""
	[ $l = base ] && V=-DV2_BASE && M=-DMDMG_BASE
	[ $l = r4 ] && V=-DV3_R3 && M=-DMDMG_R3
	nice -n 19 c++ $IN $V -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/v3_d2d_$l" "$here/v3_d2d.cpp"
	nice -n 19 c++ $IN $M -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/mdmg2_d2d_$l" "$here/mdmg2_d2d.cpp"
done
echo "== v3_run.py 10000 271828"
bash "$here/run-py.sh" v3_run.py 10000 271828 "$S/v3_d2d_base" "$S/v3_d2d_r2" "$S/v3_d2d_r4" || echo "v3 rc=$?"
echo "== mdmg2.py 20000 20261102"
bash "$here/run-py.sh" mdmg2.py 20000 20261102 "$S/mdmg2_d2d_base" "$S/mdmg2_d2d_r2" "$S/mdmg2_d2d_r4" || echo "mdmg2 rc=$?"
