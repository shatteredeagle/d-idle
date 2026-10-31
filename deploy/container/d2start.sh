#!/bin/sh
# Wizard start: make the admin token once, then serve the setup wizard (slots restore only after it is done).
# The token is the login for the UI; read it with `docker exec NAME cat /data/token`. 8431 is the only port.
set -eu
D=${D2BOT_DATA:-/data}
mkdir -p "$D"
if [ ! -s "$D/token" ]; then
  (umask 077; head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' > "$D/token")
  echo "d2start: created $D/token"
fi
# The container is the sandbox: only the explicit mounts exist inside it. bwrap is not in the image and could not
# build a user namespace under `--cap-drop ALL` plus docker's default seccomp profile anyway (U: not tried), so the
# launcher's own mount sandbox is switched off explicitly (override with -e D2_SANDBOX=auto if you add bwrap).
export D2_SANDBOX="${D2_SANDBOX:-0}"
# D2FLEET_SOAK=1: a supervised d2soak sidecar judges every slot (docs/nas-container.md "Running a soak on the NAS"): runs in
# $D/soak/<slot>-<time>/, shown by the dashboard strip and GET /api/v1/soak. No new port. The loop ends with this shell's
# pid (after exec that pid is d2fleet) and d2soak itself exits when --parent is gone.
SOAKARGS=""
if [ "${D2FLEET_SOAK:-}" = 1 ]; then
  mkdir -p "$D/soak"
  SOAKARGS="-soak-dir $D/soak"
  P=$$
  PORT=${D2FLEET_ADDR:-0.0.0.0:8431}; PORT=${PORT##*:}
  (
    while kill -0 "$P" 2>/dev/null; do
      d2soak sidecar -fleet "http://127.0.0.1:$PORT" -token-file "$D/token" -out "$D/soak" -parent "$P" \
        -profile "${D2FLEET_SOAK_PROFILE:-m24}" -for "${D2FLEET_SOAK_FOR:-24h}" >>"$D/soak/sidecar.log" 2>&1 || true
      sleep 10
    done
  ) >/dev/null 2>&1 </dev/null &
fi
# shellcheck disable=SC2086
exec d2fleet serve -setup -addr "${D2FLEET_ADDR:-0.0.0.0:8431}" -data "$D" -token-file "$D/token" $SOAKARGS
