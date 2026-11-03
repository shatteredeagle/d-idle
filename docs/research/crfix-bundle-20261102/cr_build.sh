#!/bin/bash
# cr_build.sh SRC BUILD TARGETS... : configure + build (Release, nice 19, -j2) on the worker host, as the bundle verifier's vb-build.sh
S=$1; B=$2; shift 2; D=~/d2d-lab/fork/src/build/_deps
cd "$S" || exit 1
[ -f "$B/build.ninja" ] || nice -n 19 cmake -S . -B "$B" -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DD2D_BUILD_TESTS=ON -DD2D_BUILD_APPS=ON -DD2D_BUILD_TOOLS=ON -DD2D_FFMPEG_PREFIX=$HOME/d2d-lab/fork/build-armorhit/ffmpeg-cd67286357d6 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DFETCHCONTENT_FULLY_DISCONNECTED=ON -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src -DD2D_BUNDLED_STORMLIB=OFF > "$B.cmake.log" 2>&1 || { tail -20 "$B.cmake.log"; exit 1; }
nice -n 19 ionice -c3 ninja -C "$B" -j2 "$@" > "$B.build.log" 2>&1; rc=$?; tail -3 "$B.build.log"; echo "rc=$rc"; exit $rc
