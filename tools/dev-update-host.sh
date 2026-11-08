#!/usr/bin/env bash
# The worker-host half of tools/dev-update.sh (docs/dev-update.md). Run it there, from the repo mirror; the wrapper does.
#   dev-update-host.sh --url URL --src DIR [--to C... | --all] [--rollback] [--fault badsig|unsigned|tamper|hash|crash|stale]
#   dev-update-host.sh --url URL --image TAG --d2d PATH [--src DIR]        make the DEV image once
#   dev-update-host.sh --pubkey                                            print the public key (one line, \n escaped)
#   dev-update-host.sh --url URL --serve start|stop|status [--ttl SECS]
# Result lines go to stdout, each prefixed `DEVUPDATE: `; progress goes to stderr.
# Dev key: ~/.config/didle-dev/update-ed25519.{key,pub} (made on first use, 0600, never printed, never the release key).
# Served directory: ~/.cache/didle-dev/www, a static tree shaped like the GitHub releases API (python3 http.server).
set -euo pipefail
cd "$(dirname "$0")/.."
here=$(pwd)
GO=${GO:-$(command -v go || echo "$HOME/.local/bin/go")}
dev=${DIDLE_DEV_DIR:-$HOME/.cache/didle-dev}
keydir=${DIDLE_DEV_KEYDIR:-$HOME/.config/didle-dev}
www=$dev/www
port=${DIDLE_DEV_PORT:-18899}
repo=shatteredeagle/d-idle # the path the client asks for (selfupdate.DefaultRepo); the server only imitates its releases list
url= src=. rollback=0 fault= image= d2d= serve= ttl=28800 all=0 pubkey=0 targets=()
while [ $# -gt 0 ]; do
	case $1 in
	--url) url=$2; shift 2 ;; --src) src=$2; shift 2 ;; --to) targets+=("$2"); shift 2 ;; --all) all=1; shift ;;
	--rollback) rollback=1; shift ;; --fault) fault=$2; shift 2 ;; --image) image=$2; shift 2 ;; --d2d) d2d=$2; shift 2 ;;
	--serve) serve=$2; shift 2 ;; --pubkey) pubkey=1; shift ;; --ttl) ttl=$2; shift 2 ;; *) echo "unknown flag $1" >&2; exit 2 ;;
	esac
done
say() { echo "DEVUPDATE: $*"; }
note() { echo "dev-update: $*" >&2; }
now() { date +%s; }
t0=$(now)
mkdir -p "$www/dl" "$www/repos/$repo" "$keydir"
chmod 700 "$keydir"
sign() { (cd "$here" && "$GO" run ./cmd/d2sign "$@"); }
key=$keydir/update-ed25519.key pub=$keydir/update-ed25519.pub

serve_up() { curl -fs -o /dev/null --max-time 3 "http://127.0.0.1:$port/" 2>/dev/null; }
case $serve in
start)
	if serve_up; then say "SERVING already on port $port"; exit 0; fi
	tools/agentrun.sh devupdate --ttl "$ttl" -- python3 -m http.server "$port" --bind 0.0.0.0 --directory "$www" >"$dev/server.log" 2>&1 &
	for _ in 1 2 3 4 5 6 7 8 9 10; do sleep 0.5; serve_up && break; done
	serve_up && say "SERVING port $port ttl ${ttl}s root $www" || { say "ERROR server did not start (see $dev/server.log)"; exit 1; }
	exit 0 ;;
stop) tools/agentreap.sh devupdate >&2 || true; say "SERVING stopped"; exit 0 ;;
status) serve_up && say "SERVING port $port" || say "NOT-SERVING"; exit 0 ;;
esac

[ -f "$key" ] || { note "making the dev signing key in $keydir"; sign keygen -dir "$keydir" -name update-ed25519 >/dev/null; }
[ "$pubkey" = 0 ] || { say "$(tr '\n' '|' <"$pub" | sed 's/|$//; s/|/\\n/g')"; exit 0; }
[ -n "$url" ] || { say "ERROR --url is required"; exit 2; }

if [ -n "$image" ]; then # the one-time image: DEV=1, the shim and this tree's bot baked in
	[ -f "$d2d" ] || { say "ERROR --d2d must be a d2d binary (the one the running image has)"; exit 2; }
	ctx=$(mktemp -d "$dev/ctx.XXXXXX")
	trap 'rm -rf "$ctx"' EXIT
	(cd "$src" && PREBUILT=$d2d nice -n 19 deploy/d2d-container/context.sh "$ctx") >&2
	# the legacy builder runs every stage even with D2D_SRC=prebuilt: with a prebuilt d2d the source stage becomes a stub
	python3 -I - "$ctx/Dockerfile" <<'PY'
import re, sys
p = sys.argv[1]
s = open(p).read()
s = re.sub(r"(FROM \$\{UBUNTU\} AS d2d-build\n).*?(?=# stage 1b)", r"\1RUN mkdir -p /out\n\n", s, flags=re.S)
open(p, "w").write(s)
PY
	nice -n 19 docker build -q --build-arg D2D_SRC=prebuilt --build-arg DEV=1 --build-arg APP_VERSION=0.0.0 -t "$image" "$ctx" >&2
	say "IMAGE $image built in $(( $(now) - t0 ))s (DEV=1)"
	exit 0
fi

last=$dev/last-version
next_version() { # increasing even inside one second
	local n
	n=$(date -u +%y%m%d%H%M%S)
	if [ -f "$last" ] && [ "$n" -le "$(cat "$last")" ]; then n=$(( $(cat "$last") + 1 )); fi
	echo "$n" >"$last"
	echo "$n"
}

# real releases only (a directory with a .fault marker is a drill leftover)
kept() { for d in $(ls -1 "$www/dl" | sort); do [ -e "$www/dl/$d/.fault" ] || echo "$d"; done | tail -n "${1:-5}"; }

list_releases() { # list_releases VERSION... : the static document the client reads
	python3 - "$www" "$repo" "$url" "$@" <<'PY'
import json, os, sys
www, repo, url, vers = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4:]
rels = []
for v in vers:
    d = os.path.join(www, "dl", v)
    names = sorted(n for n in os.listdir(d) if not n.startswith("."))
    rels.append({"tag_name": "app-v0.0." + v, "draft": False, "prerelease": False,
                 "assets": [{"name": n, "browser_download_url": "%s/dl/%s/%s" % (url, v, n)} for n in names]})
out = os.path.join(www, "repos", repo, "releases")
with open(out + ".tmp", "w") as f:
    json.dump(rels, f)
os.replace(out + ".tmp", out)
PY
}

publish_real() { # publish_real VERSION : build the bot from $src into dl/VERSION
	local v=$1 stage
	stage=$(mktemp -d "$dev/stage.XXXXXX")
	mkdir -p "$stage/bin" "$www/dl/$v"
	(cd "$src" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 nice -n 19 "$GO" build -trimpath -ldflags "-X main.version=0.0.$v" -o "$stage/bin/" \
		./cmd/d2bot ./cmd/d2dsoak ./cmd/d2dview ./cmd/d2dsetup ./deploy/d2d-container/d2dping)
	install -m 0755 "$src/deploy/d2d-container/entrypoint.sh" "$stage/bin/entrypoint.sh"
	tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner -C "$stage/bin" -cf - . | gzip -n >"$www/dl/$v/didle-bin-v0.0.$v-linux-amd64.tar.gz"
	rm -rf "$stage"
}

make_manifest() { # make_manifest VERSION SEQ NOTES : MANIFEST.json beside the tarball in dl/VERSION
	local v=$1 seq=$2 notes=$3 d=$www/dl/$1 f
	f=$(ls "$d"/didle-bin-*.tar.gz | head -1)
	sign manifest -dir "$d" -version "0.0.$v" -seq "$seq" -issued "$(date -u +%Y-%m-%dT%H:%M:%SZ)" -channel dev -notes "$notes" \
		-min-image 1 -bot-api 1 -d2d-ctl 1 -data-schema 1 -src-bot "dev:$v" -src-d2d none -file "$(basename "$f")=bin" -out "$d/MANIFEST.json" >/dev/null
}
sign_with() { sign sign -key "$1" -in "$www/dl/$2/MANIFEST.json" >/dev/null; }

if [ "$rollback" = 0 ]; then
	for d in $(ls -1 "$www/dl"); do [ ! -e "$www/dl/$d/.fault" ] || rm -rf "${www:?}/dl/$d"; done
	v=$(next_version)
	if [ -z "$fault" ]; then
		note "building from $src as 0.0.$v"
		publish_real "$v"
		make_manifest "$v" "$v" "dev build $(git -C "$src" log -1 --format=%h 2>/dev/null || echo tree)"
		sign_with "$key" "$v"
		sign verify -pub "$pub" -in "$www/dl/$v/MANIFEST.json" -dir "$www/dl/$v" >/dev/null
		for old in $(kept 999 | head -n -5); do rm -rf "${www:?}/dl/$old"; done
		list_releases $(kept 5)
		note "published 0.0.$v in $(( $(now) - t0 ))s"
	else
		prev=$(kept 1)
		[ -n "$prev" ] || { say "ERROR --fault needs a real release published first"; exit 1; }
		if [ "$fault" = stale ]; then # an older, validly signed release is served alone: a rollback attack
			old=$(kept 99 | head -1)
			[ "$old" != "$prev" ] || { say "ERROR --fault stale needs two published releases"; exit 1; }
			list_releases "$old"
			note "serving only the older 0.0.$old"
			v=$old
		else
			mkdir -p "$www/dl/$v"
			touch "$www/dl/$v/.fault"
			cp "$www/dl/$prev"/didle-bin-*.tar.gz "$www/dl/$v/"
			if [ "$fault" = crash ]; then # a correctly signed build whose launcher dies at once: the shim must roll it back
				tmp=$(mktemp -d "$dev/crash.XXXXXX")
				tar -xzf "$www/dl/$v"/didle-bin-*.tar.gz -C "$tmp"
				printf '#!/bin/sh\necho "FAULT crash" >&2\nexit 1\n' >"$tmp/entrypoint.sh"
				tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner -C "$tmp" -cf - . | gzip -n >"$(ls "$www/dl/$v"/didle-bin-*.tar.gz)"
				rm -rf "$tmp"
			fi
			make_manifest "$v" "$v" "FAULT $fault"
			case $fault in
			crash) sign_with "$key" "$v" ;;
			badsig) # a well-formed manifest signed by a key the container does not trust
				other=$(mktemp -d "$dev/other.XXXXXX")
				sign keygen -dir "$other" -name x >/dev/null
				sign_with "$other/x.key" "$v"
				rm -rf "$other" ;;
			unsigned) ;; # no .sig file at all
			tamper) # signed, then one byte of the notes changed
				sign_with "$key" "$v"
				sed -i 's/FAULT tamper/FAULT tamPer/' "$www/dl/$v/MANIFEST.json" ;;
			hash) # signed correctly, but the tarball differs from the signed hash
				sign_with "$key" "$v"
				printf x >>"$www/dl/$v"/didle-bin-*.tar.gz ;;
			*) say "ERROR unknown fault $fault"; exit 2 ;;
			esac
			list_releases "$v"
			note "published FAULT $fault as 0.0.$v"
		fi
	fi
	if [ "${#targets[@]}" = 0 ] && [ "$all" = 0 ]; then say "PUBLISHED 0.0.$v in $(( $(now) - t0 ))s"; exit 0; fi
fi

if [ "$all" = 1 ]; then
	mapfile -t targets < <(docker ps --filter label=io.didle.dev=1 --format '{{.Names}}')
	[ "${#targets[@]}" -gt 0 ] || { say "ERROR no running DEV container (label io.didle.dev=1)"; exit 1; }
fi

running_of() { docker exec "$1" sh -c 'sed -n "s/.*\"running\": \"\(.*\)\".*/\1/p" /data/didle/image.json' 2>/dev/null | head -1; }
wait_running() { # wait_running C VERSION : the shim restarted the bot process tree on that release
	local c=$1 v=$2 i
	for i in $(seq 1 120); do
		if docker exec "$c" sh -c "grep -l -a 'DIDLE_RELEASE=$v' /proc/[0-9]*/environ 2>/dev/null | grep -qv \"/proc/\$\$/\"" 2>/dev/null &&
			[ "$(running_of "$c")" = "$v" ]; then
			return 0
		fi
		sleep 0.5
	done
	return 1
}

rc=0
for c in "${targets[@]}"; do
	s0=$(now)
	if ! docker inspect -f '{{.State.Running}}' "$c" 2>/dev/null | grep -q true; then
		say "ERROR $c is not running"
		rc=1
		continue
	fi
	if [ "$rollback" = 1 ]; then
		out=$(docker exec "$c" /opt/didle/boot rollback-now 2>&1 || true)
		set -- $out
		if [ "${1:-}" = ROLLBACK ]; then
			if wait_running "$c" "$3"; then say "ROLLED-BACK $c $2 -> $3 in $(( $(now) - s0 ))s"; else say "ERROR $c: restart on $3 not seen"; rc=1; fi
		else
			say "ERROR $c: $out"
			rc=1
		fi
		continue
	fi
	out=$(docker exec "$c" /opt/didle/boot check-now 2>&1 || true)
	set -- $out
	case ${1:-} in
	UPDATE)
		if wait_running "$c" "$3"; then say "UPDATED $c $2 -> $3 in $(( $(now) - t0 ))s"
		else
			why=$(docker exec "$c" sh -c 'tr -d "\n" </data/didle/rollback.json 2>/dev/null' | sed -n 's/.*"reason": *"\([^"]*\)".*/\1/p')
			say "ERROR $c: $3 did not stay up and the shim went back to $(running_of "$c") (${why:-no reason recorded})"
			rc=1
		fi ;;
	CURRENT) say "CURRENT $c $2 (nothing newer)" ;;
	REJECTED) shift; say "REJECTED $c: $*"; rc=3 ;;
	*) say "ERROR $c: $out"; rc=1 ;;
	esac
done
exit $rc
