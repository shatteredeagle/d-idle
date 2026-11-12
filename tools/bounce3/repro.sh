#!/bin/sh
# Throwaway headless d2d of a play image + a driver (docs/bounce3.md). Usage: repro.sh IMAGE SEED DRIVER [ARGS]; the
# word SOCK in the arguments becomes the control socket. Examples (on the worker host):
#   repro.sh d2dbot:devp4 3 ./kickrepro -sock SOCK          (the bot's own controller, Go)
#   repro.sh d2dbot:devp4 3 python3 -I repro_kick.py SOCK   (the old and the --safe placement, Python)
# Mounts the game data read-only; never touches the sandbox container or its volume; d2d exits after 300 s anyway.
set -eu
img=$1 seed=$2; shift 2
dir=$(mktemp -d /tmp/bounce3.XXXXXX); chmod 777 "$dir"
name=bounce3-repro-$$
docker run -d --rm --name "$name" --network none -v "$HOME/d2d-lab/data:/data:ro" -v "$dir:/run/b3" -e HOME=/run/b3 \
	--entrypoint sh "$img" -c "mkdir -p /run/b3/.config/d2d/save; exec timeout 300 /opt/d2d/d2d --headless --no-save --lab-verbs --seed $seed --data /data --devctl /run/b3/s.sock --start-screen ingame --start-class 3 --start-name Bq" >/dev/null
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -rf "$dir"' EXIT
i=0; until [ -S "$dir/s.sock" ] || [ $i -ge 100 ]; do sleep 0.2; i=$((i+1)); done
sleep 4
cmd=""; for a in "$@"; do [ "$a" = SOCK ] && a=$dir/s.sock; cmd="$cmd '$a'"; done
eval "$cmd"
