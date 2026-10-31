#!/bin/sh
# Smoke test of the d2d container; run ON WORKER (never on the head box: docker build is blocked there). Nothing is pushed or deployed.
#   tools/d2d-container-smoke.sh [-k]            (-k keeps the image; the container and volume are always removed)
# Env: FORK=~/d2d-lab/fork  REF=lab5  MPQS=~/d2d-lab/data (read-only copy of the user's MPQs)  IMAGE=d2dbot:smoke
#      CTX=scratch context dir  SKIP_BUILD=1 (reuse IMAGE)  BUILD_JOBS=4  PREBUILT=path (skip the source stage)
# Steps: build, run `d2d` mode (create character), ping, state, save, restart, the character persisted.
set -eu
repo=$(cd "$(dirname "$0")/../.." && pwd)
image=${IMAGE:-d2dbot:smoke}
mpqs=${MPQS:-$HOME/d2d-lab/data}
ctx=${CTX:-${TMPDIR:-/tmp}/d2d-container-ctx}
name=d2dsmoke-$$
vol=$name-data
keep=0
[ "${1:-}" != -k ] || keep=1
ok=0
log() { echo "smoke: $*"; }
fail() { echo "smoke: FAIL: $*" >&2; docker logs --tail 30 "$name" >&2 2>&1 || true; exit 1; }
cleanup() {
	docker rm -f "$name" >/dev/null 2>&1 || true
	docker volume rm "$vol" >/dev/null 2>&1 || true
	[ "$keep" = 1 ] || [ -n "${SKIP_BUILD:-}" ] || docker image rm "$image" >/dev/null 2>&1 || true
	[ "$ok" = 1 ] || log "not ok"
}
trap cleanup EXIT

[ -f "$mpqs/d2data.mpq" ] || fail "no d2data.mpq in $mpqs"
if [ -z "${SKIP_BUILD:-}" ]; then
	"$repo/deploy/d2d-container/context.sh" "$ctx" "${REF:-lab5}" || fail "context"
	src=build
	[ -z "${PREBUILT:-}" ] || src=prebuilt
	t0=$(date +%s)
	docker build --build-arg BUILD_JOBS="${BUILD_JOBS:-4}" --build-arg D2D_SRC=$src -t "$image" "$ctx" || fail "docker build"
	log "build $(($(date +%s) - t0)) s, image $(docker image inspect -f '{{.Size}}' "$image" | awk '{printf "%.0f MB", $1/1e6}')"
fi

# MPQs read-only, saves in a named volume, no network, no capabilities
run() { # run MODE-ARGS...
	docker run -d --name "$name" --network none --cap-drop ALL --security-opt no-new-privileges:true \
		--memory 3g --cpus 2 -v "$mpqs":/game:ro -v "$vol":/data \
		-e D2D_MODE=lockstep -e D2D_NAME=Csmoke -e D2D_CLASS=1 -e D2D_SEED=3 "$image" d2d >/dev/null
}
ctl() { docker exec "$name" /opt/d2lod/d2dping -sock /run/d2d/d2d.sock.ctl "$@"; }
view() { docker exec "$name" /opt/d2lod/d2dping "$@"; }
waitping() {
	i=0
	while [ $i -lt 90 ]; do
		view ping >/dev/null 2>&1 && return 0
		sleep 1
		i=$((i + 1))
	done
	return 1
}

run
waitping || fail "ping (read-only tap) did not answer within 90 s"
log "ping: $(view ping | head -1)"
sleep 3
st=$(ctl state) || fail "state"
log "state: $(echo "$st" | head -2 | tr '\n' ' ' | cut -c1-160)"
ctl save >/dev/null || fail "save verb"
docker exec "$name" ls /data/config/d2d/save/ | grep -q 'Csmoke' || fail "no Csmoke save in the volume"
log "create + save: $(docker exec "$name" ls /data/config/d2d/save/ | tr '\n' ' ')"
health=$(docker inspect -f '{{.State.Health.Status}}' "$name" 2>/dev/null || echo none)
log "healthcheck status: $health"

docker stop -t 30 "$name" >/dev/null
docker rm "$name" >/dev/null
run
waitping || fail "ping after restart"
sleep 3
ctl key Return >/dev/null || fail "key Return on the select screen"
i=0
until ctl dump player 2>/dev/null | grep -q Csmoke; do
	i=$((i + 1))
	[ $i -lt 40 ] || fail "character Csmoke not loaded after restart"
	sleep 1
done
log "persist: character Csmoke loaded after restart"
ok=1
log "PASS"
