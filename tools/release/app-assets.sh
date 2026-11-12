#!/usr/bin/env bash
# Build the unsigned app release asset set (docs/self-update-release.md).
#   tools/release/app-assets.sh --version 0.4.0 --seq 17 --issued 2026-11-20T00:00:00Z \
#       [--source-text FILE: use FILE as SOURCE.txt instead of the default text]
#       --out DIR --d2d-bin PATH_TO_D2D [--overlay DIR] [--notes TEXT] [--src-bot REF] [--src-d2d REF] ...
# Output in DIR: one tar.gz per component, SOURCE.txt, MANIFEST.json (unsigned).
# Tarballs are reproducible (sorted names, zero mtime, root owner, gzip -n).
# Needs GNU tar. Signing is a separate job: the approval-gated one in .github/workflows/app-release.yml.
set -euo pipefail

version= seq= issued= out= d2d= overlay= notes= srcbot= srcd2d= minimage=1 botapi=1 d2dctl=1 dataschema=1 channel=stable srctext=
while [ $# -gt 0 ]; do
	case "$1" in
	--version) version=$2 ;; --seq) seq=$2 ;; --issued) issued=$2 ;; --out) out=$2 ;;
	--d2d-bin) d2d=$2 ;; --overlay) overlay=$2 ;; --notes) notes=$2 ;;
	--src-bot) srcbot=$2 ;; --source-text) srctext=$2 ;; --src-d2d) srcd2d=$2 ;; --channel) channel=$2 ;;
	--min-image) minimage=$2 ;; --bot-api) botapi=$2 ;; --d2d-ctl) d2dctl=$2 ;; --data-schema) dataschema=$2 ;;
	*) echo "unknown flag $1" >&2; exit 2 ;;
	esac
	shift 2
done
for v in version seq issued out d2d srcbot srcd2d; do
	[ -n "${!v}" ] || { echo "--${v/d2d/d2d-bin} is required" >&2; exit 2; }
done
[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "version must be x.y.z" >&2; exit 2; }
[ -f "$d2d" ] || { echo "d2d binary not found: $d2d" >&2; exit 2; }

repo=$(cd "$(dirname "$0")/../.." && pwd)
mkdir -p "$out"
out=$(cd "$out" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

pack() { # pack TARBALL DIR
	tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner -C "$2" -cf - . | gzip -n >"$1"
}

mkdir -p "$stage/bin" "$stage/d2d" "$stage/data"
(cd "$repo" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-X main.version=$version" -o "$stage/bin/" \
	./cmd/d2bot ./cmd/d2dsoak ./cmd/d2dview ./cmd/d2dsetup ./deploy/d2d-container/d2dping)
install -m 0755 "$repo/deploy/d2d-container/entrypoint.sh" "$stage/bin/entrypoint.sh" # the release carries its mode logic (design 3.9)
install -m 0755 "$d2d" "$stage/d2d/d2d"
if [ -n "$overlay" ]; then cp -r "$overlay"/. "$stage/data/"; else : >"$stage/data/.keep"; fi

pack "$out/didle-bin-v$version-linux-amd64.tar.gz" "$stage/bin"
pack "$out/didle-d2d-v$version-linux-amd64.tar.gz" "$stage/d2d"
pack "$out/didle-data-v$version.tar.gz" "$stage/data"

if [ -n "$srctext" ]; then
	[ -f "$srctext" ] || { echo "source text not found: $srctext" >&2; exit 2; }
	cp "$srctext" "$out/SOURCE.txt"
else
cat >"$out/SOURCE.txt" <<TXT
d.idle app release $version
bot source: https://github.com/shatteredeagle/d-idle at $srcbot
d2d source (GPL-3.0-or-later): https://github.com/shatteredeagle/d2d-didle at $srcd2d
The corresponding source of every binary in this release is available at the two
locations above; docs/tester-licence.md section 4 holds the written offer.
TXT
fi

(cd "$repo" && go run ./cmd/d2sign manifest -dir "$out" -version "$version" -seq "$seq" -issued "$issued" \
	-channel "$channel" -notes "$notes" -min-image "$minimage" -bot-api "$botapi" -d2d-ctl "$d2dctl" -data-schema "$dataschema" \
	-src-bot "$srcbot" -src-d2d "$srcd2d" \
	-file "didle-bin-v$version-linux-amd64.tar.gz=bin" \
	-file "didle-d2d-v$version-linux-amd64.tar.gz=d2d" \
	-file "didle-data-v$version.tar.gz=data" \
	-out "$out/MANIFEST.json")
(cd "$repo" && go run ./cmd/d2sign summary -in "$out/MANIFEST.json")
