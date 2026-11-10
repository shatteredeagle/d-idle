#!/usr/bin/env bash
# Scratch drill for tools/dev-sandbox-host.sh (run on the worker host): own container, volume and archive root, no game, never the play sandbox.
# Needs an image with tar (default d2dbot:devp1; IMG=...).
set -x -e
c=d2play-scratchreset
export DEVSANDBOX_NOWAIT=1 DEVSANDBOX_ARCHIVE_ROOT=$HOME/play-archive-scratchtest
docker rm -f $c >/dev/null 2>&1 || true; docker volume rm d2play-scratch-data >/dev/null 2>&1 || true; rm -rf "$DEVSANDBOX_ARCHIVE_ROOT"
docker volume create d2play-scratch-data >/dev/null
docker run --rm --entrypoint sh -v d2play-scratch-data:/data ${IMG:-d2dbot:devp1} -c "mkdir -p /data/instances/main/state /data/instances/main/d2d/config/d2d/save /data/didle; echo s >/data/instances/main/d2d/config/d2d/save/Doofer.d2s; echo '{}' >/data/instances/main/state/goldfarm-ui.json; echo keep >/data/access.json; printf \"D2D_NAME='Doofer'\nD2D_CLASS='2'\nD2BOT_GOALS='campaign:normal_act1'\n\" >/data/wizard.env"
docker create --name $c --entrypoint sleep -v d2play-scratch-data:/data ${IMG:-d2dbot:devp1} 3600 >/dev/null; docker start $c >/dev/null
tools/dev-sandbox-host.sh reset --to $c --class sorceress --name Freshy --dry-run
tools/dev-sandbox-host.sh reset --to $c --class sorceress --name Freshy || echo rc=$?
tools/dev-sandbox-host.sh reset --to $c --class nope --yes || echo rc=$?
tools/dev-sandbox-host.sh reset --to $c --class sorceress --name Freshy --yes
docker run --rm --entrypoint sh -v d2play-scratch-data:/data ${IMG:-d2dbot:devp1} -c "cat /data/wizard.env; ls -R /data | head; cat /data/access.json"
tools/dev-sandbox-host.sh list --to $c
tools/dev-sandbox-host.sh restore --to $c --from last --yes
docker run --rm --entrypoint sh -v d2play-scratch-data:/data ${IMG:-d2dbot:devp1} -c "cat /data/wizard.env; find /data -type f"
tools/dev-sandbox-host.sh list --to $c
docker rm -f $c; docker volume rm d2play-scratch-data
