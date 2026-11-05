#!/bin/sh
# Keeps a credential-free, read-only mirror of the public repo and serves its board.
# Mirror = shallow single-branch clone; every SYNC_INTERVAL seconds fetch + reset --hard
# + clean, so anything written through the web UI is discarded. Backlog.md is restarted
# when the mirror head moves (it reads from git, not only from the work tree: U on caching).
set -u
: "${REPO_URL:?}" "${BRANCH:?}" "${MIRROR:?}" "${SYNC_INTERVAL:=300}"
BACKLOG_PORT=6421
log() { echo "[board $(date -u +%H:%M:%SZ)] $*"; }

sync_once() {
  if [ ! -d "$MIRROR/.git" ]; then
    rm -rf "$MIRROR"; mkdir -p "$MIRROR"
    git clone --quiet --depth 1 --single-branch --branch "$BRANCH" "$REPO_URL" "$MIRROR" \
      || { log "clone failed"; return 1; }
    return 0
  fi
  git -C "$MIRROR" fetch --quiet --depth 1 origin "$BRANCH" \
    && git -C "$MIRROR" reset --quiet --hard FETCH_HEAD \
    && git -C "$MIRROR" clean --quiet -fdx \
    || { log "fetch failed, keeping old mirror"; return 1; }
}
head_of() { git -C "$MIRROR" rev-parse HEAD 2>/dev/null || echo none; }

start_backlog() {
  (cd "$MIRROR" && exec backlog browser --no-open --non-interactive --port "$BACKLOG_PORT") &
  BPID=$!
}

until sync_once; do log "retry in 30s"; sleep 30; done
mkdir -p /tmp/nx_body /tmp/nx_proxy /tmp/nx_fcgi /tmp/nx_uwsgi /tmp/nx_scgi
nginx -c /etc/nginx/board.conf -g 'daemon off;' &
NPID=$!
start_backlog
last=$(head_of); log "serving $last"
trap 'kill "$BPID" "$NPID" 2>/dev/null; exit 0' TERM INT

while :; do
  sleep "$SYNC_INTERVAL" & wait $!
  sync_once
  now=$(head_of)
  if [ "$now" != "$last" ] || ! kill -0 "$BPID" 2>/dev/null; then
    kill "$BPID" 2>/dev/null; wait "$BPID" 2>/dev/null
    start_backlog; last=$now; log "restarted on $now"
  fi
  kill -0 "$NPID" 2>/dev/null || { log "nginx died"; exit 1; }
done
