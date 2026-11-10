#!/usr/bin/env bash
# The worker-host half of tools/dev-sandbox.sh (docs/dev-reset.md). Subcommands are the cmd_* functions below.
# Never deletes the data volume or touches the game mount; state is moved into an archive outside the volume first, and the
# archive is checked before anything is removed. Never reads access.json or watch.token (they are neither archived nor changed).
# Test knobs: DEVSANDBOX_ARCHIVE_ROOT (default ~/play-archive-<suffix of the container name>), DEVSANDBOX_NOWAIT=1 (do not wait for the hero).
set -euo pipefail
cd "$(dirname "$0")/.."
tablesjson=cmd/d2dsetup/tables.json # class id -> d2d index, the one table (the wizard reads the same file)
sub=${1:?subcommand}; shift
c= class= name= yes=0 dry=0 from=last
while [ $# -gt 0 ]; do
	case $1 in
	--to) c=$2; shift 2 ;; --class) class=$2; shift 2 ;; --name) name=$2; shift 2 ;; --from) from=$2; shift 2 ;;
	--yes) yes=1; shift ;; --dry-run) dry=1; shift ;; *) echo "dev-sandbox: unknown argument $1" >&2; exit 2 ;;
	esac
done
die() { echo "dev-sandbox: $*" >&2; exit 1; }
note() { echo "dev-sandbox: $*" >&2; }
[ -n "$c" ] || die "--to CONTAINER is required"
docker inspect "$c" >/dev/null 2>&1 || die "no container named $c"
vol=$(docker inspect "$c" --format '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}}')
[ -n "$vol" ] || die "$c has no named volume at /data"
image=$(docker inspect "$c" --format '{{.Config.Image}}')
aroot=${DEVSANDBOX_ARCHIVE_ROOT:-$HOME/play-archive-${c#d2play-}}
# a short-lived helper on the container's own image, volume mounted, no network; the game mount is never passed
vrun() { docker run --rm -i --network none --entrypoint sh -v "$vol:/data" "$image" -c "$@"; }
stopc() { note "stopping $c (the bot saves the hero first, up to 60 s)"; docker stop -t 60 "$c" >/dev/null; }
startc() { docker start "$c" >/dev/null; }

# ---- archive: state.tgz (instances/, config/, wizard.env) + meta + sha256, mode 700 dir
make_archive() { # $1 = label suffix; prints the archive dir
	local stamp dir
	stamp=$(date +%Y%m%d-%H%M%S)$1
	dir=$aroot/$stamp
	mkdir -p "$dir"; chmod 700 "$aroot" "$dir"
	vrun 'cd /data; set --; for p in instances config wizard.env; do [ -e "$p" ] && set -- "$@" "$p"; done; [ $# -gt 0 ] || exit 3; tar -czf - "$@"' >"$dir/state.tgz" || die "archive failed (kept $dir)"
	tar -tzf "$dir/state.tgz" >"$dir/files.txt" || die "archive unreadable (kept $dir)"
	[ -s "$dir/files.txt" ] || die "archive is empty (kept $dir)"
	(cd "$dir" && sha256sum state.tgz >state.tgz.sha256)
	{ echo "container=$c"; echo "volume=$vol"; echo "image=$image"; echo "created=$(date -Is)"; vrun 'cat /data/wizard.env 2>/dev/null' | sed 's/^/wizard./' || true; } >"$dir/meta.txt"
	echo "$dir"
}
wipe_state() { vrun 'cd /data && rm -rf instances config wizard.env && echo wiped'; }
wait_hero() { # up to 240 s for the save file of $1
	[ "${DEVSANDBOX_NOWAIT:-0}" = 0 ] || return 0
	local i
	for i in $(seq 1 48); do
		docker exec "$c" sh -c "ls /data/instances/*/d2d/config/d2d/save/'$1'.d2s" >/dev/null 2>&1 && return 0
		sleep 5
	done
	return 1
}
report() { # version and URL of the running sandbox
	local ver origin
	ver=$(docker exec "$c" sh -c 'sed -n "s/.*\"running\"[^\"]*\"\([^\"]*\)\".*/\1/p" /data/didle/state.json 2>/dev/null | head -1' 2>/dev/null || true)
	origin=$(docker inspect "$c" --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^D2BOT_WATCH_ORIGIN=//p' | head -1)
	echo "VERSION ${ver:-unknown}"; echo "URL ${origin:-unknown}"
}
resolve() { # archive arg -> dir
	local a=$1
	[ "$a" != last ] || a=$(ls -1 "$aroot" 2>/dev/null | tail -1)
	[ -n "$a" ] || die "no archives under $aroot"
	[ -d "$a" ] || a=$aroot/$a
	[ -f "$a/state.tgz" ] || die "no archive $a"
	echo "$a"
}

cmd_list() {
	[ -d "$aroot" ] || { echo "no archives (root $aroot)"; return 0; }
	for d in "$aroot"/*/; do [ -f "$d/state.tgz" ] && echo "$(basename "$d")  $(du -h "$d/state.tgz" | cut -f1)  $(grep -h '^wizard.D2D_NAME' "$d/meta.txt" | tr -d "'")"; done
}

cmd_reset() {
	local idx cur curname curclass
	curname=$(vrun "sed -n \"s/^D2D_NAME='\\(.*\\)'/\\1/p\" /data/wizard.env 2>/dev/null" || true)
	curclass=$(vrun "sed -n \"s/^D2D_CLASS='\\(.*\\)'/\\1/p\" /data/wizard.env 2>/dev/null" || true)
	[ -n "$name" ] || name=${curname:-D2dbot}
	[[ $name =~ ^[A-Za-z][A-Za-z_-]{1,14}$ ]] || die "--name must be 2 to 15 letters (also - and _), starting with a letter"
	if [ -n "$class" ]; then
		idx=$(python3 -I - "$tablesjson" "$class" <<'PY'
import json, sys
t = json.load(open(sys.argv[1]))["classes"]
m = {c["id"]: c["d2d_index"] for c in t}
if sys.argv[2] not in m:
    sys.exit("unknown class %r (known: %s)" % (sys.argv[2], ", ".join(sorted(m))))
print(m[sys.argv[2]])
PY
		) || exit 1
	else idx=${curclass:?no class in wizard.env; pass --class}; fi
	echo "PLAN reset $c: archive instances/ config/ wizard.env of volume $vol to $aroot/<stamp>, remove them from the volume, hero $name class index $idx (was ${curname:-none}/${curclass:-none}), restart"
	echo "PLAN kept in place: access.json watch.token didle/ cache/ wizard.json, and the game mount"
	if [ "$dry" = 1 ]; then echo "DRY-RUN nothing changed"; return 0; fi
	[ "$yes" = 1 ] || die "refusing without --yes (or use --dry-run)"
	stopc
	cur=$(make_archive "")
	echo "ARCHIVE $cur"
	wipe_state >/dev/null
	printf "D2D_NAME='%s'\nD2D_CLASS='%s'\nD2BOT_GOALS='campaign:normal_act1'\n" "$name" "$idx" | vrun 'cat >/data/wizard.env'
	startc
	wait_hero "$name" && echo "HERO $name created" || echo "HERO $name not seen after 240 s (check: docker logs $c)"
	report
	echo "RESTORE tools/dev-sandbox.sh restore --to $c --from $(basename "$cur") --yes"
}

cmd_restore() {
	local a pre
	a=$(resolve "$from")
	(cd "$a" && sha256sum -c state.tgz.sha256 >/dev/null) || die "checksum of $a does not match"
	echo "PLAN restore $c from $a (the current state is archived first as ...-pre-restore)"
	if [ "$dry" = 1 ]; then echo "DRY-RUN nothing changed"; return 0; fi
	[ "$yes" = 1 ] || die "refusing without --yes (or use --dry-run)"
	stopc
	pre=$(make_archive "-pre-restore")
	echo "ARCHIVE $pre"
	wipe_state >/dev/null
	vrun 'cd /data && tar -xzf -' <"$a/state.tgz"
	startc
	local n; n=$(sed -n "s/^wizard.D2D_NAME='\(.*\)'/\1/p" "$a/meta.txt")
	[ -z "$n" ] || { wait_hero "$n" && echo "HERO $n loaded" || echo "HERO $n not seen after 240 s"; }
	report
	echo "UNDO tools/dev-sandbox.sh restore --to $c --from $(basename "$pre") --yes"
}

declare -F "cmd_$sub" >/dev/null || die "unknown subcommand $sub"
"cmd_$sub"
