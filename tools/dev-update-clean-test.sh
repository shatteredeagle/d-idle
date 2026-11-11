#!/usr/bin/env bash
# Offline regression test (TASK devupdateclean): a file deleted between two deploys must be gone from the worker copy.
# Uses a fake labrun and a scratch repo; no worker host, no container, never the play sandbox.
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd)
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
mkdir -p "$t/repo/tools" "$t/bin" "$t/host"
cp "$here/tools/dev-update.sh" "$here/tools/env.sh" "$t/repo/tools/"
# fake labrun: emulates the real one (rsync of --send paths WITHOUT --delete, then the command in the host copy)
cat >"$t/bin/labrun" <<'F'
#!/usr/bin/env bash
shift # machine name
sends=()
while [ "$1" != -- ]; do [ "$1" = --send ] && sends+=("$2"); shift 2; done
shift
for s in "${sends[@]+"${sends[@]}"}"; do mkdir -p "$FAKEHOST/$s"; rsync -rlpt "$s/" "$FAKEHOST/$s/"; done
cd "$FAKEHOST"
if [ "$1" = tools/dev-update-host.sh ]; then echo "SRC: $(find .devupdate-src -type f | sort | tr '\n' ' ')" >>"$FAKEHOST.log"; echo "DEVUPDATE: UPDATED fake"; else "$@"; fi
F
chmod +x "$t/bin/labrun"
cd "$t/repo"; git init -q; git config user.email t@t; git config user.name t
mkdir -p pkg; echo a >pkg/keep.go; echo x >pkg/stale.go
git add -A; git commit -qm A; A=$(git rev-parse HEAD)
git rm -q pkg/stale.go; echo b >pkg/new.go; git add -A; git commit -qm B; B=$(git rev-parse HEAD)
export PATH="$t/bin:$PATH" FAKEHOST="$t/host" D2_LABRUN_HOST=fake D2_WORKER_HOST=w D2_BOT_DIR="$t/repo"
tools/dev-update.sh --commit "$A" --to c >/dev/null
grep -q 'stale.go' "$t/host.log" || { echo "FAIL: deploy A lacks stale.go"; exit 1; }
tools/dev-update.sh --commit "$B" --to c >/dev/null
last=$(tail -1 "$t/host.log")
[ ! -e "$t/host/.devupdate-src/pkg/stale.go" ] || { echo "FAIL: stale.go survived on the host copy"; exit 1; }
case $last in *stale.go*) echo "FAIL: build input has stale.go: $last"; exit 1 ;; esac
case $last in *pkg/new.go*pkg/keep.go* | *pkg/keep.go*pkg/new.go*) ;; *) echo "FAIL: build input incomplete: $last"; exit 1 ;; esac
echo "ok: dev-update mirrors deletions"
