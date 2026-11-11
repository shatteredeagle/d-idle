#!/bin/bash
# verifyb2 (the worker host): build one tree of the batch 2 check from the verifier's own clone.
#   vb2_build.sh NAME COMMIT [ninja targets...]   (default target: game)
# Tree ~/d2d-lab/vb2/wt-NAME (a worktree of the verifier's clone, not of the fork checkout),
# build ~/d2d-lab/vb2/build-NAME (Release, D2D_WERROR=ON, the CI warning set), nice 19, -j2.
set -u
NAME=$1; C=$2; shift 2
V=$HOME/d2d-lab/vb2
T=$V/wt-$NAME; B=$V/build-$NAME
D=$HOME/d2d-lab/src/build/_deps
[ -d "$T" ] || git -C "$V/src" worktree add -q --detach "$T" "$C" || exit 1
git -C "$T" log -1 --format='tree %h %s'
mkdir -p "$B"
if [ ! -f "$B/build.ninja" ]; then
  nice -n 19 cmake -S "$T" -B "$B" -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=ON \
    -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
    -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src -DCMAKE_EXPORT_COMPILE_COMMANDS=ON > "$B/../cmake-$NAME.log" 2>&1 \
    || { tail -20 "$B/../cmake-$NAME.log"; exit 1; }
fi
[ $# -gt 0 ] || set -- game
nice -n 19 ninja -C "$B" -j2 "$@" > "$V/build-$NAME.log" 2>&1
rc=$?
tail -4 "$V/build-$NAME.log"
echo "build $NAME rc=$rc"
exit $rc
