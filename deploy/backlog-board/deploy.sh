#!/bin/bash
# Build the backlog-board image on the NAS and (re)create the container from its NAS template.
# Idempotent. Run from a checkout:  BOARD_IP=<br0-ip> deploy/backlog-board/deploy.sh
# Host values come from tools/env.sh (D2_NAS_HOST, local.env); the br0 address is only given here.
# NPM, DNS, Authelia and the Folder View entry are one-time steps, see the backlog-board doc in docs/process.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
. "$here/../../tools/env.sh"
: "${BOARD_IP:?set BOARD_IP to a free br0 address}"
KEY="$HOME/.ssh/id_ed25519_nas"
SSH="ssh -i $KEY -o IdentitiesOnly=yes root@$D2_NAS_HOST"
TPL=/boot/config/plugins/dockerMan/templates-user/my-backlog-board.xml

# Template: the repo copy has a <br0-ip> placeholder and a comment line for it.
sed -e "s#<br0-ip>#$BOARD_IP#" -e '/Replace <br0-ip>/d' "$here/my-backlog-board.xml" \
  | $SSH "cat > $TPL"

# Ship the build context (Dockerfile pins and checksum-verifies the Backlog.md release).
ctx=$($SSH 'mktemp -d /tmp/bb-build.XXXXXX')
tar -C "$here" -cf - Dockerfile entrypoint.sh nginx.conf | $SSH "tar -C $ctx -xf -"

# Build, then recreate the container from the template (not compose, not a bare docker run).
$SSH "bash -s $ctx" <<'REMOTE'
set -euo pipefail
ctx=$1; trap 'rm -rf "$ctx"' EXIT
docker build -q -t backlog-board:1.55.3 "$ctx" >/dev/null
mkdir -p /mnt/user/appdata/backlog-board/data && chown 99:100 /mnt/user/appdata/backlog-board/data
bash /mnt/user/appdata/_*/tpl-apply.sh backlog-board
for i in $(seq 1 40); do
  [ "$(docker inspect -f '{{.State.Health.Status}}' backlog-board 2>/dev/null)" = healthy ] && { echo "backlog-board healthy"; exit 0; }
  sleep 3
done
echo "backlog-board not healthy after 120 s" >&2; exit 1
REMOTE
