#!/bin/bash
# hirereader (the worker host): the whole purse differential. Needs hrd_build.sh run first.
#   tools/emu/run-on-worker.sh --ttl 2400 hrd_run.sh [N] > /tmp/d2agent-hirereader/pay.txt 2>&1
# For each of hire / revive and seeds 1..4: game.exe's FUN_00576d90 (and FUN_006637b0 for revive) against d2d's
# rules::hire / rules::resurrect_merc on N cases (default 25000); prints the counts and the first differences.
set -u
here=$(cd "$(dirname "$0")" && pwd)
N=${1:-25000}
OUT=${D2_SCRATCH:-/tmp/d2agent-hirereader}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
UP=${UP:-$HOME/d2d-lab/src}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
total=0; bad=0
for mode in hire revive; do
	for seed in 1 2 3 4; do
		(cd "$UP/tools/emu" && nice "$PY" "$here/hrd_pay.py" $mode $N $seed) > $OUT/g_${mode}_$seed.txt 2> $OUT/g_${mode}_$seed.err
		nice "$OUT/hrd_d2d_test" $mode $N $seed > $OUT/d_${mode}_$seed.txt
		cases=$(grep -c ': ' $OUT/g_${mode}_$seed.txt)
		diffs=$(diff $OUT/g_${mode}_$seed.txt $OUT/d_${mode}_$seed.txt | grep -c '^<')
		echo "$mode seed $seed: cases $cases differ $diffs"
		total=$((total + cases)); bad=$((bad + diffs))
		diff $OUT/g_${mode}_$seed.txt $OUT/d_${mode}_$seed.txt | head -6
	done
done
echo "TOTAL cases $total differ $bad"
