#!/bin/bash
# skdefenders (the worker host): the whole differential for the given seeds (default 1 and 7), 100,000 random
# cases each plus the boundary grid. Build first (skdef_build.sh).
#   tools/emu/run-on-worker.sh --ttl 3000 skdef_all.sh [N] [SEED ...]
here=$(cd "$(dirname "$0")" && pwd)
N=${1:-100000}
shift
SEEDS=${*:-1 7}
for s in $SEEDS; do
	echo "######## seed $s"
	bash "$here/skdef_run.sh" "$N" "$s" || exit 1
	bash "$here/skdef_diff.sh" "$s" || exit 1
	bash "$here/skdef_rule.sh" "$s" "$N" || exit 1
done
