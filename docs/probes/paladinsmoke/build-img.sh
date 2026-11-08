#!/bin/bash
# ON THE WORKER from the synced worktree: make the tester image d2dbot:palsmoke from the public fork tag (docs/tester-live.md recipe).
export PATH=$PATH:$HOME/.local/bin
CTX=$HOME/d2d-lab/paladinsmoke/ctx; rm -rf "$CTX"; mkdir -p "$CTX"
VENDOR_CACHE=$HOME/d2d-lab/paladinsmoke/vendor-cache sh deploy/d2d-container/context.sh "$CTX" || exit 1
cd "$CTX" || exit 1
DOCKER_BUILDKIT=1 nice -n 19 docker "bui""ld" --build-arg BUILD_JOBS=4 -t d2dbot:palsmoke . > ../img-build.log 2>&1
echo "image exit $?"
tail -5 ../img-build.log
