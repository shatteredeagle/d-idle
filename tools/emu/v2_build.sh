#!/bin/bash
# verifydmg2: compile tools/emu/v2_d2d.cpp against the rules headers of base and r2 (exported by v2_trees.sh), nice 19, one at a time.
#   tools/emu/run-on-worker.sh v2_build.sh
set -eu
S=${D2_SCRATCH:?}
here=$(cd "$(dirname "$0")" && pwd)
for l in base r2; do
	IN=""
	for d in components/rules components/d2s components/txt components/log; do IN="$IN -I$S/t_$l/$d"; done
	DEF=""; [ $l = base ] && DEF=-DV2_BASE
	nice -n 19 c++ $IN $DEF -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/v2_d2d_$l" "$here/v2_d2d.cpp"
	echo "built v2_d2d_$l"
done
