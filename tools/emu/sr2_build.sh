#!/bin/sh
# SORCREST2: build the d2d rim probe (sr2_test.cpp) next to the SORCREST scratch build on the worker host. 2 jobs, nice.
set -e
SRC=$HOME/d2d-lab/pr/sorcrest-d2d
OUT=$HOME/d2d-lab/pr/sorcrest-out
cp "$(dirname "$0")/sr2_test.cpp" $SRC/tests/game/test_sr2.cpp
cd $OUT/build
nice cmake . > /dev/null
nice ninja -j2 test_sr2
ls -la tests/test_sr2
