#!/usr/bin/env bash
# Run a d2idle:dev container on a lab worker with the standard hardening and volumes.
#   tools/container-run.sh NAME [docker run flags...] -- d2container-args...
# Env: IMG (d2idle:dev), DATA_VOL (d2cg-data), GAME_DIR ($HOME/d2lod-container-data/game, mounted read-only at /game),
#      UI_PORT (8431: published on 127.0.0.1 only; empty = publish nothing), DOCKER_NETWORK (default bridge; "none" = offline).
# The wizard UI needs the default bridge network to be reachable (`--network none` cannot publish ports). The only
# port is the UI; the token is `docker exec NAME cat /data/token`. A native docker run line is in docs/container.md.
set -eu
NAME=$1; shift
IMG=${IMG:-d2idle:dev}; DATA_VOL=${DATA_VOL:-d2cg-data}; GAME_DIR=${GAME_DIR:-$HOME/d2lod-container-data/game}
UI_PORT=${UI_PORT-8431}; NET=${DOCKER_NETWORK:-bridge}
FLAGS=()
while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do FLAGS+=("$1"); shift; done
[ "${1:-}" = "--" ] && shift
PUB=()
[ -n "$UI_PORT" ] && [ "$NET" != none ] && PUB=(-p "127.0.0.1:$UI_PORT:8431")
exec docker run --name "$NAME" --label d2lod.container=1 \
  --network "$NET" ${PUB[@]+"${PUB[@]}"} --cap-drop ALL --security-opt no-new-privileges \
  --shm-size 256m --memory 3g --cpus 2 --stop-timeout 90 \
  -v "$DATA_VOL":/data -v "$GAME_DIR":/game:ro \
  ${FLAGS[@]+"${FLAGS[@]}"} "$IMG" "$@"
