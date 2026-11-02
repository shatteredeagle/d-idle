#!/bin/bash
# verifynmlevel (the worker host): clean builds of the four trees, one at a time, -j2 (nothing else of this agent runs meanwhile), niced, own ccache.
#   vnl_build.sh base|head       : tests on (the suites + libgame for the probe)
#   vnl_build.sh labbase|labhead : the d2d binary with the lab patch set (det scenarios), copied to $F/vnl-bin/d2d.<base|head>
#   vnl_build.sh all             : the four in that order
set -o pipefail
F=$HOME/d2d-lab/fork
S=$F/src/build/_deps
t=$1; T=$F/wt-vnl-$t; B=$F/build-vnl-$t
export CCACHE_DIR=/tmp/d2agent-verifynmlevel/ccache
mkdir -p "$CCACHE_DIR" "$F/vnl-bin"
common=(-G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DCMAKE_CXX_COMPILER_LAUNCHER=ccache -DCMAKE_C_COMPILER_LAUNCHER=ccache
  "-DFETCHCONTENT_SOURCE_DIR_CLI11=$S/cli11-src" "-DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$S/sentry_native-src"
  "-DFETCHCONTENT_SOURCE_DIR_STORMLIB=$S/stormlib-src" -DD2D_BUNDLED_STORMLIB=OFF)
case $t in
base|head)
  [ -f "$B/build.ninja" ] || cmake -S "$T" -B "$B" "${common[@]}" > "$B.cmake.log" 2>&1 || { tail -20 "$B.cmake.log"; exit 1; }
  nice -n 15 ninja -C "$B" -j2 > "$B.build.log" 2>&1; rc=$?; tail -3 "$B.build.log"; exit $rc ;;
labbase|labhead)
  [ -f "$B/build.ninja" ] || cmake -S "$T" -B "$B" "${common[@]}" -DD2D_BUILD_APPS=ON "-DD2D_FFMPEG_PREFIX=$F/build-armorhit/ffmpeg-cd67286357d6" \
    -DD2D_BUILD_TOOLS=ON -DFETCHCONTENT_FULLY_DISCONNECTED=ON -DD2D_BUILD_TESTS=OFF > "$B.cmake.log" 2>&1 || { tail -20 "$B.cmake.log"; exit 1; }
  nice -n 15 ninja -C "$B" -j2 d2d > "$B.build.log" 2>&1 || { tail -20 "$B.build.log"; exit 1; }
  cp "$B/apps/d2d/d2d" "$F/vnl-bin/d2d.${t#lab}"; sha256sum "$F/vnl-bin/d2d.${t#lab}" ;;
all)
  for x in base head labbase labhead; do echo "== $x $(date +%T)"; bash "$0" "$x" || echo "FAILED $x"; done; echo "ALLDONE $(date +%T)" ;;
esac
