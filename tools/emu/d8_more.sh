#!/bin/bash
# decomp8 (worker host): TASK-80 AC4 (Lut Gholein, level 40, recorded not fixed) and TASK-77 more seeds.
HERE=$(cd "$(dirname "$0")" && pwd)
echo "== level 40 tiles units seeds 1-30"; sh $HERE/d8_ss.sh 1-30 40 tiles units seeds 2>&1 | grep -v WARNING | cut -c1-220 | tail -12
echo "== store 201-400"; sh $HERE/d8_store.sh 201-400 2>&1 | grep -v -E "WARNING" | head -6 | cut -c1-200
