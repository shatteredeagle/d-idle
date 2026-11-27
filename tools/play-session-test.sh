#!/bin/sh
# Offline test of play-session.sh: dry-run output and the refusal without the ops checkout.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
ps=$here/play-session.sh
fail() { echo "FAIL: $1"; exit 1; }
# refusal: no ops checkout
rc=0; out=$(D2_OPS_DIR=$t/none "$ps" --dry-run start 2>&1) || rc=$?
[ "$rc" = 2 ] || fail "refusal exit $rc"
[ "$(printf '%s\n' "$out" | wc -l)" = 1 ] || fail "refusal is not one line"
case $out in *"ops repo not present"*) ;; *) fail "refusal text" ;; esac
# dry run with a fake ops checkout
mkdir "$t/ops"; : >"$t/ops/AGENTS.private.md"
out=$(D2_OPS_DIR=$t/ops D2_WORKER_HOST=w.test D2_WORKER_USER=u "$ps" --dry-run --instance i3 start)
for want in "game.sh start i3" "-ladder -watch -watch-video" "-watch-token-file .d2bot/watch-token-i3" "watch URL:  http://w.test:8891/watch" "token file: .d2bot/watch-token-i3"; do
	case $out in *"$want"*) ;; *) fail "dry-run lacks: $want" ;; esac
done
case $out in *"?t="*"?t=<token>"*|*"?t=<token>"*) ;; *) fail "token placeholder" ;; esac
# i0 and bad instance refused, stop dry run
D2_OPS_DIR=$t/ops "$ps" --dry-run --instance i0 start >/dev/null 2>&1 && fail "i0 accepted"
out=$(D2_OPS_DIR=$t/ops "$ps" --dry-run --instance i3 stop)
case $out in *"game.sh stop i3"*) ;; *) fail "stop" ;; esac
echo "play-session test ok"
