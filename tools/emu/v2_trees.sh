#!/bin/bash
# verifydmg2: export base 3e073888, first candidate 6f9ded7c and r2 b9ecb415 (components/ and tests/) from the private clone
# in $D2_SCRATCH (made by v2_git.sh) into $D2_SCRATCH/t_LABEL, then grep a pattern in all three.
#   tools/emu/run-on-worker.sh v2_trees.sh [PATTERN]
set -u
S=${D2_SCRATCH:?}
C=$S/clone
for p in base:3e073888 r1:6f9ded7c r2:b9ecb415; do
	l=${p%%:*}; h=${p#*:}
	[ -d "$S/t_$l" ] || { mkdir -p "$S/t_$l" && git -C "$C" archive "$h" components tests | tar -x -C "$S/t_$l"; }
done
[ $# -ge 1 ] && grep -rn -- "$1" "$S/t_base/components" "$S/t_r2/components" | sed "s#$S/##"
exit 0
