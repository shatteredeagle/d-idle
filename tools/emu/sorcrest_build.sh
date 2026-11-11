#!/bin/sh
# SORCREST: build the d2d side on the worker host from a scratch copy of the SKGROUND2 checkout
# (upstream main 7d5b3a5c + the Blizzard shard commits, irrelevant to bursts). 2 jobs, nice.
set -e
SRC=$HOME/d2d-lab/pr/sorcrest-d2d
OUT=$HOME/d2d-lab/pr/sorcrest-out
[ -d $SRC ] || { mkdir -p $SRC && (cd $HOME/d2d-lab/pr/skground2 && tar --exclude=./build --exclude=./.git -cf - .) | tar -xf - -C $SRC; }
cp "$(dirname "$0")/sk_rest_test.cpp" $SRC/tests/game/test_sk_rest.cpp
sed '1a // test-links: game' "$(dirname "$0")/nec_test.cpp" > $SRC/tests/game/test_sk_rest_calc.cpp
mkdir -p $OUT/build
cd $OUT/build
[ -f build.ninja ] || nice cmake -G Ninja -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER_LAUNCHER=ccache $SRC
nice ninja -j2 test_sk_rest test_sk_rest_calc
ls -la tests/test_sk_rest tests/test_sk_rest_calc
