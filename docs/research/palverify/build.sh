#!/bin/bash
# build.sh: configure and build base and head scratch copies from clean, probe dropped into tests/game (agent palverify).
# Runs on the worker host under agentrun; log in $R/build.log.
R=$HOME/d2d-lab/palverify; D=$HOME/d2d-lab/src/build/_deps
log() { echo "$(date +%T) $*" >> $R/build.log; }
for w in base head; do
  cp $R/h/test_palprobe.cpp $R/$w/tests/game/test_palprobe.cpp
  flags=""; [ $w = head ] && flags="-DPV_HEAD"
  rm -rf $R/build-$w
  (cd $R/$w && cmake -S . -B $R/build-$w -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DCMAKE_CXX_FLAGS="$flags" \
     -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $R/build-$w.cmake.log 2>&1)
  log "cmake $w rc=$?"
  nice ninja -C $R/build-$w -j4 tests/test_palprobe tools/mpq-cat/mpq-cat > $R/build-$w.build.log 2>&1
  log "build $w rc=$?"
done
log done
