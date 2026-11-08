#!/bin/bash
# decomp5 (worker host): patch the scratch checkout's test_game with the d5_*.py patch file then build test_game
set -u
T=$HOME/d2d-lab/wt-decomp5; B=$HOME/d2d-lab/build-decomp5
[ -n "${1:-}" ] && python3 "$1"
nice -n 19 ninja -C $B -j2 test_game > $B.build2.log 2>&1
echo "build rc=$?"; grep -E "error|FAILED" $B.build2.log | head -20; tail -1 $B.build2.log
