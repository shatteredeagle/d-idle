#!/bin/bash
# pipeline.sh: my builds one after the other, -j2 nice (agent verifyplayerroll). Log: pipeline.log
R=~/d2d-lab/verifyplayerroll; D=~/d2d-lab/fork; cd $R
log() { echo "$(date +%T) $*" >> $R/pipeline.log; }
b() { local w=$1; shift; log "build $w $*"; nice ninja -C $R/build-$w -j2 "$@" > $R/build-$w.$(echo "$@" | tr ' /' '__' | cut -c1-40).log 2>&1; log "build $w rc=$?"; }
b head tests/test_rollprobe tests/test_combat tests/test_rules tests/test_game_rules tests/test_game tests/test_missile_hit
b base tests/test_rollprobe tests/test_combat tests/test_rules tests/test_game_rules tests/test_game
if [ ! -f $R/build-labhead/build.ninja ]; then
  (cd $R/labhead && cmake -S . -B $R/build-labhead -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/build-cheats3/_deps/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/build-click/_deps/sentry_native-src -DD2D_BUNDLED_STORMLIB=OFF > $R/build-labhead.cmake.log 2>&1); log "cmake labhead rc=$?"
fi
b labhead d2d
b head all
b base all
log done
