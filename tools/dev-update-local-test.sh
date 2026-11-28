#!/usr/bin/env bash
# Offline test (TASK devupdatefix): local mode keeps the staged copy alive to the host call, remote mode still cleans+sends,
# the first line names the target host, and the stage dir is removed on exit. Fake labrun, no docker, no worker host.
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd)
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
mkdir -p "$t/repo/tools" "$t/bin" "$t/host"
cp "$here/tools/dev-update.sh" "$here/tools/env.sh" "$t/repo/tools/"
printf '#!/bin/sh\necho "docker stub $*" >>"%s/docker.log"\n' "$t" >"$t/bin/docker"; chmod +x "$t/bin/docker"
# fake labrun: logs every call; for the host script it records whether the --src dir exists at that moment
cat >"$t/bin/labrun" <<'F'
#!/usr/bin/env bash
echo "labrun $*" >>"$FAKELOG"
shift
sends=()
while [ "$1" != -- ]; do [ "$1" = --send ] && sends+=("$2"); shift 2; done
shift
for s in "${sends[@]+"${sends[@]}"}"; do mkdir -p "$FAKEHOST/$s"; rsync -rlpt "$s/" "$FAKEHOST/$s/"; done
if [ "$1" = tools/dev-update-host.sh ]; then
	while [ $# -gt 0 ]; do [ "$1" = --src ] && src=$2; shift; done
	case $src in /*) ;; *) src=$FAKEHOST/$src ;; esac
	[ -f "$src/pkg/keep.go" ] && echo "SRC-OK $src" >>"$FAKELOG" || echo "SRC-MISSING $src" >>"$FAKELOG"
	echo "DEVUPDATE: UPDATED fake"
elif [ "$1" = rm ]; then (cd "$FAKEHOST" && "$@"); else "$@"; fi
F
chmod +x "$t/bin/labrun"
cd "$t/repo"; git init -q; git config user.email t@t; git config user.name t
mkdir -p pkg; echo a >pkg/keep.go; git add -A; git commit -qm A; A=$(git rev-parse HEAD)
export PATH="$t/bin:$PATH" FAKEHOST="$t/host" FAKELOG="$t/log" D2_LABRUN_HOST=lab D2_WORKER_HOST=w
fail() { echo "FAIL: $*"; cat "$t/log"; exit 1; }

# local mode: no clean, no send, absolute --src that exists at the host call, stage gone afterwards, first line names the host
D2_SANDBOX_HOST=sbx D2_LABRUN_LOCAL=1 tools/dev-update.sh --commit "$A" --to c >"$t/out"
[ "$(head -1 "$t/out")" = "dev-update: host=sbx sandbox-host=sbx labrun-host=lab mode=local" ] || fail "first line: $(head -1 "$t/out")"
grep -q 'UPDATED fake' "$t/out" || fail "no result line"
grep -q '^SRC-OK /' "$t/log" || fail "local: staged dir missing or not absolute"
grep -q 'rm -rf\|--send' "$t/log" && fail "local mode cleaned or sent"
ls -d .devupdate-src.* 2>/dev/null && fail "stage dir left behind"

# remote mode: clean then send, relative src, target is the labrun host without D2_SANDBOX_HOST
: >"$t/log"
D2_LABRUN_LOCAL=0 tools/dev-update.sh --commit "$A" --to c >"$t/out"
[ "$(head -1 "$t/out")" = "dev-update: host=lab sandbox-host=lab labrun-host=lab mode=remote" ] || fail "first line: $(head -1 "$t/out")"
grep -q '^labrun lab -- rm -rf \.devupdate-src\.' "$t/log" || fail "remote: no clean"
grep -q '^labrun lab --send \.devupdate-src\.' "$t/log" || fail "remote: no send"
grep -q '^SRC-OK' "$t/log" || fail "remote: host copy missing"

# --serve goes to the sandbox host
: >"$t/log"
D2_SANDBOX_HOST=sbx D2_LABRUN_LOCAL=0 tools/dev-update.sh --serve status >/dev/null
grep -q '^labrun sbx -- tools/dev-update-host.sh .*--serve status' "$t/log" || fail "serve not on the sandbox host"
echo "ok: dev-update local/remote mode"
