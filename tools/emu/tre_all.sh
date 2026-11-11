#!/bin/bash
# treasure card: the five upstream drop differentials against the pristine fork build, one after the other
# (rolls over a seed range the upstream proof did not use, then tables, items, objects, stands).
HERE=$(cd "$(dirname "$0")" && pwd)
for args in "100001-200000" "tables" "items 1-40000" "objects 1-20000" "stands 1-20000"; do
    echo "== tre_diff.py $args"
    bash "$HERE/tre_run.sh" tre_diff.py $args 2>&1 | tail -8
done
echo "== done"
