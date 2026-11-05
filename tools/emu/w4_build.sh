#!/bin/bash
# verifydmg3: compile tools/emu/w4_d2d.cpp against the rules headers of base, r2 and r3 (exported by w4_git.sh), nice 19, one at a time.
#   tools/emu/run-on-worker.sh w4_build.sh
set -eu
S=${D2_SCRATCH:?}
here=$(cd "$(dirname "$0")" && pwd)
for l in base r2 r3 r4; do
	IN=""
	for d in components/rules components/d2s components/txt components/log; do IN="$IN -I$S/t_$l/$d"; done
	DEF=""; [ $l = base ] && DEF=-DW4_BASE; [ $l = r2 ] && DEF=-DW4_R2; [ $l = r3 ] && DEF=-DW4_R3
	nice -n 19 c++ $IN $DEF -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/w4_d2d_$l" "$here/w4_d2d.cpp"
	echo "built w4_d2d_$l"
done
